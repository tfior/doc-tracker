// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package seed fills a development database with two example cases. It writes
// through the HTTP API, in process, as the Seed Data user, so every record goes
// through the same validation and activity logging as a user's own. The
// account is locked again when the seed finishes.
package seed

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/tfior/doc-tracker/internal/app"
	"github.com/tfior/doc-tracker/internal/users"
)

// cases are the seed cases, each with the function that builds it. A case
// whose title is already present is skipped, so the seed can be run more than
// once.
var cases = []struct {
	title string
	build func(*builder)
}{
	{"Rossi → Martini", rossi},
	{"Esposito → Keller", esposito},
}

// CaseTitles are the titles of the seed cases.
var CaseTitles = func() []string {
	titles := make([]string, len(cases))
	for i, c := range cases {
		titles[i] = c.title
	}
	return titles
}()

// UserEmail is the Seed Data user's email. The .invalid domain is reserved and
// never receives mail.
const UserEmail = "seed@doc-tracker.invalid"

// Run adds every seed case that is not already present.
func Run(ctx context.Context, db *sql.DB) (err error) {
	var missing []int
	for i, sc := range cases {
		var exists bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cases WHERE title = $1)`, sc.title).Scan(&exists); err != nil {
			return fmt.Errorf("check for case %q: %w", sc.title, err)
		}
		if !exists {
			missing = append(missing, i)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	svc := users.NewService(users.NewStore(db))
	u, password, err := unlock(ctx, svc)
	if err != nil {
		return err
	}
	// The account is locked however the seed ends, a panic included.
	defer func() { err = errors.Join(err, svc.Lock(ctx, u.ID)) }()

	c := &client{handler: app.NewHandler(db)}
	if err := c.login(password); err != nil {
		return err
	}
	for _, i := range missing {
		b := c.newCase(cases[i].title)
		cases[i].build(b)
		if b.err != nil {
			return fmt.Errorf("case %q is incomplete: %w; recreate the development database and run the seed again", cases[i].title, b.err)
		}
	}
	c.do("DELETE", "/api/v1/auth/session", nil, nil)
	return c.err
}

// unlock finds or creates the Seed Data user and gives them a new random
// password, used once by this run and never stored. An account with the seed's
// email that the seed did not create is left alone.
func unlock(ctx context.Context, svc *users.Service) (*users.User, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, "", err
	}
	password := hex.EncodeToString(b)
	u, err := svc.GetByEmail(ctx, UserEmail)
	if errors.Is(err, users.ErrNotFound) {
		u, err = svc.Create(ctx, UserEmail, "Seed", "Data", password)
		return u, password, err
	}
	if err != nil {
		return nil, "", err
	}
	if u.FirstName != "Seed" || u.LastName != "Data" {
		return nil, "", fmt.Errorf("the account %s is not the Seed Data user; the seed will not change it", UserEmail)
	}
	return u, password, svc.SetPassword(ctx, u.ID, password)
}

// client sends requests to the application in process, as a logged-in user.
// After the first failed request it sends nothing more and keeps that error,
// so a case can be built as a straight list of steps.
type client struct {
	handler  http.Handler
	cookies  []*http.Cookie
	statuses map[string]string // "phase/label" → status ID
	err      error
}

func (c *client) login(password string) error {
	rec := c.send("POST", "/api/v1/auth/session", map[string]string{"email": UserEmail, "password": password})
	if rec.Code != http.StatusOK {
		return fmt.Errorf("log in as the seed user: status %d: %s", rec.Code, rec.Body.String())
	}
	c.cookies = rec.Result().Cookies()
	c.statuses = map[string]string{}
	for _, phase := range []string{"official_copy", "amendment", "apostille", "translation"} {
		var list []struct{ ID, Label string }
		c.do("GET", "/api/v1/document-statuses?phase="+phase, nil, &list)
		for _, s := range list {
			c.statuses[phase+"/"+s.Label] = s.ID
		}
	}
	return c.err
}

func (c *client) send(method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, ck := range c.cookies {
		req.AddCookie(ck)
	}
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	return rec
}

// do sends a request that must succeed, decoding the response into out.
func (c *client) do(method, path string, body, out any) {
	if c.err != nil {
		return
	}
	rec := c.send(method, path, body)
	if rec.Code >= 300 {
		c.err = fmt.Errorf("%s %s: status %d: %s", method, path, rec.Code, rec.Body.String())
		return
	}
	if out != nil {
		if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
			c.err = fmt.Errorf("%s %s: decode: %w", method, path, err)
		}
	}
}

// record is what the seed reads back from a created record.
type record struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

// builder adds the records of one case.
type builder struct {
	*client
	base string // the case's API path
}

func (c *client) newCase(title string) *builder {
	var r record
	c.do("POST", "/api/v1/cases", map[string]any{"title": title}, &r)
	return &builder{client: c, base: "/api/v1/cases/" + r.ID}
}

// fields is a request body.
type fields map[string]any

func q(value, qualifier string) fields {
	return fields{"value": value, "qualifier": qualifier}
}

func exact(value string) fields { return q(value, "exact") }

func (b *builder) person(first, last string, f fields) string {
	body := fields{"first_name": first, "last_name": last}
	for k, v := range f {
		body[k] = v
	}
	var r record
	b.do("POST", b.base+"/people", body, &r)
	return r.ID
}

func (b *builder) event(personID, eventType string, f fields) string {
	body := fields{"person_id": personID, "event_type": eventType}
	for k, v := range f {
		body[k] = v
	}
	var r record
	b.do("POST", b.base+"/life-events", body, &r)
	return r.ID
}

// vital returns the person's birth or death event, created by the person API.
func (b *builder) vital(personID, eventType string) string {
	var list struct {
		Items []struct {
			ID        string `json:"id"`
			PersonID  string `json:"person_id"`
			EventType string `json:"event_type"`
		} `json:"items"`
	}
	b.do("GET", b.base+"/life-events", nil, &list)
	for _, le := range list.Items {
		if le.PersonID == personID && le.EventType == eventType {
			return le.ID
		}
	}
	if b.err == nil {
		b.err = fmt.Errorf("no %s event for person %s", eventType, personID)
	}
	return ""
}

// phases are a document's statuses by phase, as labels; a phase left out
// keeps its default.
type phases map[string]string

func (b *builder) document(personID, eventID, docType, title string, f fields, p phases) {
	body := fields{"person_id": personID, "document_type": docType, "title": title}
	if eventID != "" {
		body["life_event_id"] = eventID
	}
	for k, v := range f {
		body[k] = v
	}
	var r record
	b.do("POST", b.base+"/documents", body, &r)
	if len(p) == 0 || b.err != nil {
		return
	}
	update := fields{"version": r.Version}
	for phase, label := range p {
		id, ok := b.statuses[phase+"/"+label]
		if !ok {
			b.err = fmt.Errorf("no %q status for phase %s", label, phase)
			return
		}
		update[phase+"_status_id"] = id
	}
	b.do("PATCH", b.base+"/documents/"+r.ID, update, nil)
}

func (b *builder) claimLine(rootID, status, notes string) {
	b.do("POST", b.base+"/claim-lines", fields{"root_person_id": rootID, "status": status, "notes": notes}, nil)
}

// Phase statuses used often.
var (
	ready      = phases{"official_copy": "Complete", "amendment": "Not Required", "apostille": "Complete", "translation": "Complete"}
	readyItaly = phases{"official_copy": "Complete", "amendment": "Not Required", "apostille": "Not Required", "translation": "Not Required"}
	notStarted = phases{}
	checked    = fields{"is_verified": true}
)

// with returns the fields of a and b together.
func with(a, b fields) fields {
	out := fields{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// rossi is a line through a great-grandfather born in Italy, which the 2025
// citizenship law no longer allows.
func rossi(b *builder) {
	giuseppe := b.person("Giuseppe", "Rossi", fields{
		"birth_date": exact("1875-03-15"), "birth_place": "Palermo, Sicily, Italy",
		"death_date": exact("1952-11-03"), "death_place": "New York, NY",
	})
	antonio := b.person("Antonio", "Rossi", fields{
		"birth_date": exact("1905-09-22"), "birth_place": "New York, NY",
		"death_date": exact("1978-05-20"), "death_place": "New York, NY",
		"parent_ids": []string{giuseppe},
	})
	carlo := b.person("Carlo", "Rossi", fields{
		"birth_date": exact("1932-08-11"), "birth_place": "Brooklyn, NY",
		"death_date": exact("2015-02-27"), "death_place": "Queens, NY",
		"parent_ids": []string{antonio},
	})
	sofia := b.person("Sofia", "Martini", fields{
		"birth_date": exact("1962-03-08"), "birth_place": "Queens, NY",
		"parent_ids": []string{carlo},
	})

	gMarriage := b.event(giuseppe, "marriage", fields{
		"event_date": exact("1902-06-10"), "event_place": "New York, NY",
		"spouse_name": "Maria Ferretti", "spouse_birth_place": "Naples, Italy",
	})
	gNaturalization := b.event(giuseppe, "naturalization", fields{
		"event_date": exact("1915-07-04"), "event_place": "New York, NY",
	})
	aMarriage := b.event(antonio, "marriage", fields{
		"event_date": exact("1930-04-18"), "event_place": "Brooklyn, NY",
		"spouse_name": "Maria Conti", "spouse_birth_place": "Sicily, Italy",
	})
	cMarriage := b.event(carlo, "marriage", fields{
		"event_date": exact("1958-09-14"), "event_place": "Queens, NY",
		"spouse_name": "Linda Walsh", "spouse_birth_place": "Boston, MA",
	})
	sMarriage := b.event(sofia, "marriage", fields{
		"event_date": exact("1990-07-22"), "event_place": "New York, NY",
		"spouse_name": "Marco Martini",
	})

	b.document(giuseppe, b.vital(giuseppe, "birth"), "birth_certificate", "Italian Birth Certificate", with(checked, fields{
		"issuing_authority": "Comune di Palermo", "issue_date": "2024-03-12", "registration_date": "1875-03-17",
		"recorded_given_name": "Giuseppe", "recorded_surname": "Rossi",
		"recorded_birth_date": "1875-03-15", "recorded_birth_place": "Palermo",
	}), readyItaly)
	b.document(giuseppe, gMarriage, "marriage_certificate", "Marriage Certificate", with(checked, fields{
		"issuing_authority": "New York City Clerk", "recorded_given_name": "Giuseppe", "recorded_surname": "Rossi",
		"recorded_birth_place": "Palermo, Italy",
	}), ready)
	b.document(giuseppe, gNaturalization, "naturalization", "US Naturalization Certificate", fields{
		"issuing_authority": "U.S. District Court, S.D.N.Y.", "issue_date": "1915-07-04",
		"recorded_given_name": "Joseph", "recorded_surname": "Rossi", "recorded_birth_date": "1875-03-16",
		"notes": "Records the name as Joseph and the birth date as 16 March 1875; the birth certificate says Giuseppe, 15 March.",
	}, phases{"official_copy": "Complete", "amendment": "Not Required"})
	b.document(giuseppe, b.vital(giuseppe, "death"), "death_certificate", "Death Certificate", nil, notStarted)

	b.document(antonio, b.vital(antonio, "birth"), "birth_certificate", "US Birth Certificate", with(checked, fields{
		"issuing_authority": "NYC Department of Health", "issue_date": "2024-02-20",
		"recorded_given_name": "Antonio", "recorded_surname": "Rossi",
		"recorded_birth_date": "1905-09-22", "recorded_birth_place": "New York",
	}), ready)
	b.document(antonio, aMarriage, "marriage_certificate", "Marriage Certificate", fields{
		"recorded_given_name": "Antonio", "recorded_surname": "Rossi",
	}, phases{"official_copy": "Complete", "amendment": "Not Required", "apostille": "Complete", "translation": "Ready for Review"})
	b.document(antonio, b.vital(antonio, "death"), "death_certificate", "Death Certificate", fields{
		"recorded_given_name": "Antonio", "recorded_surname": "Russo",
		"notes": "Records the surname as Russo; an amendment may be needed.",
	}, phases{"official_copy": "Complete", "amendment": "Researching"})

	b.document(carlo, b.vital(carlo, "birth"), "birth_certificate", "US Birth Certificate", with(checked, fields{
		"issuing_authority": "NYC Department of Health", "issue_date": "2024-02-20",
		"recorded_given_name": "Carlo", "recorded_surname": "Rossi",
		"recorded_birth_date": "1932-08-11", "recorded_birth_place": "Brooklyn",
	}), phases{"official_copy": "Complete", "amendment": "Not Required", "apostille": "Sent"})
	b.document(carlo, cMarriage, "marriage_certificate", "Marriage Certificate", nil, phases{"official_copy": "Requested"})
	b.document(carlo, b.vital(carlo, "death"), "death_certificate", "Death Certificate", nil, notStarted)

	b.document(sofia, b.vital(sofia, "birth"), "birth_certificate", "US Birth Certificate", with(checked, fields{
		"issuing_authority": "NYC Department of Health", "issue_date": "2024-01-15",
		"recorded_given_name": "Sofia", "recorded_surname": "Rossi",
		"recorded_birth_date": "1962-03-08", "recorded_birth_place": "Queens",
	}), ready)
	b.document(sofia, sMarriage, "marriage_certificate", "Marriage Certificate", nil, phases{"official_copy": "Researching"})

	b.claimLine(giuseppe, "ineligible",
		"Under the 2025 citizenship law, a claim needs a parent or grandparent born in Italy. "+
			"Sofia's nearest Italian-born ancestor is her great-grandfather.")
}

// esposito is a line through grandparents born in Italy: through the
// grandmother it holds; through the grandfather it is broken by his
// naturalization before their daughter's birth.
func esposito(b *builder) {
	paolo := b.person("Paolo", "Esposito", fields{
		"birth_date": exact("1934-11-20"), "birth_place": "Naples, Campania, Italy",
	})
	lucia := b.person("Lucia", "Bianchi", fields{
		"birth_date": exact("1938-05-02"), "birth_place": "Avellino, Campania, Italy",
	})
	anna := b.person("Anna", "Esposito", fields{
		"birth_date": exact("1963-01-30"), "birth_place": "Boston, MA",
		"parent_ids": []string{paolo, lucia},
	})
	emily := b.person("Emily", "Keller", fields{
		"birth_date": exact("1991-04-17"), "birth_place": "Cambridge, MA",
		"parent_ids": []string{anna},
	})

	marriage := b.event(paolo, "marriage", fields{
		"event_date": exact("1959-08-15"), "event_place": "Naples, Italy", "spouse_person_id": lucia,
	})
	pNaturalization := b.event(paolo, "naturalization", fields{
		"event_date": q("1961", "about"), "event_place": "Boston, MA",
	})
	lNaturalization := b.event(lucia, "naturalization", fields{
		"event_date": exact("1975-06-12"), "event_place": "Boston, MA",
	})
	aMarriage := b.event(anna, "marriage", fields{
		"event_date": exact("1988-10-08"), "event_place": "Boston, MA",
		"spouse_name": "David Keller", "spouse_birth_date": q("1960", "about"), "spouse_birth_place": "Hartford, CT",
	})

	b.document(paolo, b.vital(paolo, "birth"), "birth_certificate", "Italian Birth Certificate", with(checked, fields{
		"issuing_authority": "Comune di Napoli", "issue_date": "2024-05-06",
		"recorded_given_name": "Paolo", "recorded_surname": "Esposito",
		"recorded_birth_date": "1934-11-20", "recorded_birth_place": "Napoli",
	}), readyItaly)
	b.document(paolo, marriage, "marriage_certificate", "Italian Marriage Certificate", with(checked, fields{
		"issuing_authority": "Comune di Napoli", "issue_date": "2024-05-06",
		"recorded_given_name": "Paolo", "recorded_surname": "Esposito",
	}), readyItaly)
	b.document(paolo, pNaturalization, "naturalization", "Naturalization Record", fields{
		"issuing_authority": "U.S. District Court, D. Mass.", "issue_date": "1961-04",
		"notes": "Naturalized before Anna's birth, so the line through Paolo is broken.",
	}, phases{"official_copy": "Researching"})

	b.document(lucia, b.vital(lucia, "birth"), "birth_certificate", "Italian Birth Certificate", fields{
		"issuing_authority":   "Comune di Avellino",
		"recorded_given_name": "Lucia", "recorded_surname": "Bianchi",
		"recorded_birth_date": "1938-05-02", "recorded_birth_place": "Avellino",
	}, phases{"official_copy": "Requested"})
	b.document(lucia, lNaturalization, "naturalization", "US Naturalization Certificate", fields{
		"issuing_authority": "U.S. District Court, D. Mass.", "issue_date": "1975-06-12",
		"recorded_given_name": "Lucy", "recorded_surname": "Esposito", "recorded_birth_date": "1938-02-05",
		"notes": "Records the birth date as 5 February 1938; the birth certificate says 2 May 1938, likely a day–month swap.",
	}, phases{"official_copy": "Complete", "amendment": "Required — Not Started"})

	b.document(anna, b.vital(anna, "birth"), "birth_certificate", "US Birth Certificate", with(checked, fields{
		"issuing_authority": "Massachusetts Registry of Vital Records", "issue_date": "2024-04-02",
		"recorded_given_name": "Anna", "recorded_surname": "Esposito",
		"recorded_birth_date": "1963-01-30", "recorded_birth_place": "Boston",
	}), phases{"official_copy": "Complete", "amendment": "Not Required", "apostille": "Sent"})
	b.document(anna, aMarriage, "marriage_certificate", "Marriage Certificate", fields{
		"recorded_given_name": "Anna", "recorded_surname": "Espósito",
		"notes": "Records the surname with an accent: Espósito.",
	}, phases{"official_copy": "Complete", "amendment": "Researching"})

	b.document(emily, b.vital(emily, "birth"), "birth_certificate", "US Birth Certificate", with(checked, fields{
		"issuing_authority": "Massachusetts Registry of Vital Records", "issue_date": "2024-04-02",
		"recorded_given_name": "Emily", "recorded_surname": "Keller",
		"recorded_birth_date": "1991-04-17", "recorded_birth_place": "Cambridge",
	}), ready)

	b.claimLine(paolo, "ineligible",
		"Paolo naturalized in about 1961, before Anna was born in 1963, so he did not pass citizenship to her.")
	b.claimLine(lucia, "eligible",
		"Lucia naturalized in 1975, after Anna's birth in 1963. Anna was born after 1948, so she inherited "+
			"through her mother. Lucia is Emily's grandparent born in Italy.")
}
