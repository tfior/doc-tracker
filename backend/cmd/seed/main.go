package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/tfior/doc-tracker/platform"
)

func main() {
	cfg := platform.LoadConfig()

	db, err := platform.OpenDatabase(cfg)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	if err := seed(db); err != nil {
		log.Fatalf("seed: %v", err)
	}
	log.Println("seed complete")
}

// seeder holds a transaction and stops executing once an error is encountered.
type seeder struct {
	tx  *sql.Tx
	err error
}

// id executes a query with a RETURNING id::text clause and returns the scanned ID.
func (s *seeder) id(query string, args ...any) string {
	if s.err != nil {
		return ""
	}
	var id string
	s.err = s.tx.QueryRow(query, args...).Scan(&id)
	return id
}

// exec executes a query with no return value.
func (s *seeder) exec(query string, args ...any) {
	if s.err != nil {
		return
	}
	_, s.err = s.tx.Exec(query, args...)
}

func seed(db *sql.DB) error {
	// Idempotency — skip if already seeded.
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM cases WHERE title = $1)`, "Rossi → Martini").Scan(&exists); err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if exists {
		log.Println("seed data already present, skipping")
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Look up system document status IDs.
	rows, err := tx.Query(`SELECT system_key::text, id::text FROM document_statuses WHERE is_system = true`)
	if err != nil {
		return fmt.Errorf("query statuses: %w", err)
	}
	statusIDs := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			rows.Close()
			return err
		}
		statusIDs[k] = v
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	pending := statusIDs["pending"]
	collected := statusIDs["collected"]
	verified := statusIDs["verified"]

	s := &seeder{tx: tx}

	// -------------------------------------------------------------------------
	// Case
	// -------------------------------------------------------------------------

	caseID := s.id(`INSERT INTO cases (title, status) VALUES ($1, 'active') RETURNING id::text`, "Rossi → Martini")

	// -------------------------------------------------------------------------
	// People
	// -------------------------------------------------------------------------

	giuseppeID := s.id(`
		INSERT INTO people (case_id, first_name, last_name, birth_date, birth_place, death_date)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
		caseID, "Giuseppe", "Rossi", "1875-03-15", "Palermo, Sicily, Italy", "1952-11-03")

	antonioID := s.id(`
		INSERT INTO people (case_id, first_name, last_name, birth_date, birth_place, death_date)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
		caseID, "Antonio", "Rossi", "1905-09-22", "New York, NY", "1978-05-20")

	carloID := s.id(`
		INSERT INTO people (case_id, first_name, last_name, birth_date, birth_place, death_date)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
		caseID, "Carlo", "Rossi", "1932-08-11", "Brooklyn, NY", "2015-02-27")

	sofiaID := s.id(`
		INSERT INTO people (case_id, first_name, last_name, birth_date, birth_place)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text`,
		caseID, "Sofia", "Martini", "1962-03-08", "Queens, NY")

	// -------------------------------------------------------------------------
	// PersonRelationships (parent-child lineage)
	// -------------------------------------------------------------------------

	s.exec(`INSERT INTO person_relationships (person_id, parent_id, case_id) VALUES ($1, $2, $3)`, antonioID, giuseppeID, caseID)
	s.exec(`INSERT INTO person_relationships (person_id, parent_id, case_id) VALUES ($1, $2, $3)`, carloID, antonioID, caseID)
	s.exec(`INSERT INTO person_relationships (person_id, parent_id, case_id) VALUES ($1, $2, $3)`, sofiaID, carloID, caseID)

	// -------------------------------------------------------------------------
	// ClaimLine and case LIRA
	// -------------------------------------------------------------------------

	s.exec(`INSERT INTO claim_lines (case_id, root_person_id, status) VALUES ($1, $2, 'confirmed')`, caseID, giuseppeID)
	s.exec(`UPDATE cases SET primary_root_person_id = $1 WHERE id = $2`, giuseppeID, caseID)

	// -------------------------------------------------------------------------
	// Life Events
	// -------------------------------------------------------------------------

	giuseppeBirthID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place)
		VALUES ($1, $2, 'birth', $3, $4) RETURNING id::text`,
		caseID, giuseppeID, "1875-03-15", "Palermo, Sicily, Italy")

	giuseppeMarriageID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place, spouse_name, spouse_birth_place)
		VALUES ($1, $2, 'marriage', $3, $4, $5, $6) RETURNING id::text`,
		caseID, giuseppeID, "1902-06-10", "New York, NY", "Maria Ferretti", "Naples, Italy")

	giuseppeDeathID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place)
		VALUES ($1, $2, 'death', $3, $4) RETURNING id::text`,
		caseID, giuseppeID, "1952-11-03", "New York, NY")

	antonioBirthID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place)
		VALUES ($1, $2, 'birth', $3, $4) RETURNING id::text`,
		caseID, antonioID, "1905-09-22", "New York, NY")

	antonioMarriageID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place, spouse_name, spouse_birth_place)
		VALUES ($1, $2, 'marriage', $3, $4, $5, $6) RETURNING id::text`,
		caseID, antonioID, "1930-04-18", "Brooklyn, NY", "Maria Conti", "Sicily, Italy")

	antonioDeathID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place)
		VALUES ($1, $2, 'death', $3, $4) RETURNING id::text`,
		caseID, antonioID, "1978-05-20", "New York, NY")

	carloBirthID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place)
		VALUES ($1, $2, 'birth', $3, $4) RETURNING id::text`,
		caseID, carloID, "1932-08-11", "Brooklyn, NY")

	carloMarriageID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place, spouse_name, spouse_birth_place)
		VALUES ($1, $2, 'marriage', $3, $4, $5, $6) RETURNING id::text`,
		caseID, carloID, "1958-09-14", "Queens, NY", "Linda Walsh", "Boston, MA")

	carloDeathID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place)
		VALUES ($1, $2, 'death', $3, $4) RETURNING id::text`,
		caseID, carloID, "2015-02-27", "Queens, NY")

	sofiaBirthID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place)
		VALUES ($1, $2, 'birth', $3, $4) RETURNING id::text`,
		caseID, sofiaID, "1962-03-08", "Queens, NY")

	sofiaMarriageID := s.id(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, event_place, spouse_name)
		VALUES ($1, $2, 'marriage', $3, $4, $5) RETURNING id::text`,
		caseID, sofiaID, "1990-07-22", "New York, NY", "Marco Martini")

	// -------------------------------------------------------------------------
	// Documents
	// -------------------------------------------------------------------------

	// Giuseppe — Italian birth certificate (verified)
	giuseppeBirthCertID := s.id(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title,
			 issuing_authority, issue_date,
			 recorded_given_name, recorded_surname,
			 is_verified, verified_at)
		VALUES ($1, $2, $3, $4, 'birth_certificate', 'Italian Birth Certificate',
			'Comune di Palermo, Italy', $5,
			'Giuseppe', 'Rossi',
			true, '2024-06-01 00:00:00+00')
		RETURNING id::text`,
		caseID, giuseppeID, giuseppeBirthID, verified, "1875-03-15")

	// Giuseppe — Marriage certificate (verified)
	giuseppeMarriageCertID := s.id(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title,
			 is_verified, verified_at)
		VALUES ($1, $2, $3, $4, 'marriage_certificate', 'Marriage Certificate',
			true, '2024-06-01 00:00:00+00')
		RETURNING id::text`,
		caseID, giuseppeID, giuseppeMarriageID, verified)

	// Giuseppe — US naturalization certificate (collected, unverified)
	// Amendment uploaded — is_verified stays false per domain rule.
	// life_event_id is NULL; no naturalization LifeEvent in this case's seed data.
	giuseppeNatCertID := s.id(`
		INSERT INTO documents
			(case_id, person_id, status_id, document_type, title,
			 issuing_authority, issue_date,
			 is_verified, notes)
		VALUES ($1, $2, $3, 'naturalization', 'US Naturalization Certificate',
			'U.S. District Court, S.D.N.Y.', $4,
			false,
			'Date of naturalization must post-date Antonio''s birth (22 Sep 1905) to preserve citizenship transmission. Amendment added middle initial "L." to recorded name.')
		RETURNING id::text`,
		caseID, giuseppeID, collected, "1915-07-04")

	// Giuseppe — Death certificate (pending)
	s.exec(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title, is_verified)
		VALUES ($1, $2, $3, $4, 'death_certificate', 'Death Certificate', false)`,
		caseID, giuseppeID, giuseppeDeathID, pending)

	// Antonio — US birth certificate (verified)
	antonioBirthCertID := s.id(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title,
			 issuing_authority, issue_date,
			 is_verified, verified_at, notes)
		VALUES ($1, $2, $3, $4, 'birth_certificate', 'US Birth Certificate',
			'NYC Department of Health', $5,
			true, '2024-06-01 00:00:00+00',
			'Proves birth on 22 Sep 1905, prior to Giuseppe''s naturalization on 4 Jul 1915.')
		RETURNING id::text`,
		caseID, antonioID, antonioBirthID, verified, "1905-09-22")

	// Antonio — Marriage certificate (verified)
	antonioMarriageCertID := s.id(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title,
			 is_verified, verified_at)
		VALUES ($1, $2, $3, $4, 'marriage_certificate', 'Marriage Certificate',
			true, '2024-06-01 00:00:00+00')
		RETURNING id::text`,
		caseID, antonioID, antonioMarriageID, verified)

	// Antonio — Death certificate (collected, unverified)
	antonioDeathCertID := s.id(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title, is_verified)
		VALUES ($1, $2, $3, $4, 'death_certificate', 'Death Certificate', false)
		RETURNING id::text`,
		caseID, antonioID, antonioDeathID, collected)

	// Carlo — US birth certificate (verified)
	carloBirthCertID := s.id(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title,
			 issuing_authority, issue_date,
			 is_verified, verified_at)
		VALUES ($1, $2, $3, $4, 'birth_certificate', 'US Birth Certificate',
			'NYC Department of Health', $5,
			true, '2024-06-01 00:00:00+00')
		RETURNING id::text`,
		caseID, carloID, carloBirthID, verified, "1932-08-11")

	// Carlo — Marriage certificate (collected, unverified)
	carloMarriageCertID := s.id(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title, is_verified)
		VALUES ($1, $2, $3, $4, 'marriage_certificate', 'Marriage Certificate', false)
		RETURNING id::text`,
		caseID, carloID, carloMarriageID, collected)

	// Carlo — Death certificate (pending)
	s.exec(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title, is_verified)
		VALUES ($1, $2, $3, $4, 'death_certificate', 'Death Certificate', false)`,
		caseID, carloID, carloDeathID, pending)

	// Sofia — US birth certificate (verified)
	sofiaBirthCertID := s.id(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title,
			 issuing_authority, issue_date,
			 is_verified, verified_at)
		VALUES ($1, $2, $3, $4, 'birth_certificate', 'US Birth Certificate',
			'NYC Department of Health', $5,
			true, '2024-06-01 00:00:00+00')
		RETURNING id::text`,
		caseID, sofiaID, sofiaBirthID, verified, "1962-03-08")

	// Sofia — Marriage certificate (pending, no attachments)
	s.exec(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title, is_verified)
		VALUES ($1, $2, $3, $4, 'marriage_certificate', 'Marriage Certificate', false)`,
		caseID, sofiaID, sofiaMarriageID, pending)

	// -------------------------------------------------------------------------
	// FileAttachments
	// Pending documents have no attachments.
	// -------------------------------------------------------------------------

	// Giuseppe — Italian birth certificate
	s.exec(`
		INSERT INTO file_attachments
			(document_id, storage_key, filename, content_type, size_bytes, is_canonical, attachment_type)
		VALUES ($1, $2, $3, 'application/pdf', $4, true, 'original')`,
		giuseppeBirthCertID, "seed/giuseppe-rossi/birth-certificate.pdf", "birth-certificate.pdf", 1572864)

	// Giuseppe — Marriage certificate
	s.exec(`
		INSERT INTO file_attachments
			(document_id, storage_key, filename, content_type, size_bytes, is_canonical, attachment_type)
		VALUES ($1, $2, $3, 'application/pdf', $4, true, 'original')`,
		giuseppeMarriageCertID, "seed/giuseppe-rossi/marriage-certificate.pdf", "marriage-certificate.pdf", 1835008)

	// Giuseppe — Naturalization certificate: original (superseded) + amendment (canonical)
	s.exec(`
		INSERT INTO file_attachments
			(document_id, storage_key, filename, content_type, size_bytes, is_canonical, attachment_type, superseded_at)
		VALUES ($1, $2, $3, 'application/pdf', $4, false, 'original', '2024-08-15 14:30:00+00')`,
		giuseppeNatCertID, "seed/giuseppe-rossi/naturalization-original.pdf", "naturalization-original.pdf", 2097152)

	s.exec(`
		INSERT INTO file_attachments
			(document_id, storage_key, filename, content_type, size_bytes, is_canonical, attachment_type)
		VALUES ($1, $2, $3, 'application/pdf', $4, true, 'amendment')`,
		giuseppeNatCertID, "seed/giuseppe-rossi/naturalization-amendment.pdf", "naturalization-amendment.pdf", 2097152)

	// Antonio — US birth certificate
	s.exec(`
		INSERT INTO file_attachments
			(document_id, storage_key, filename, content_type, size_bytes, is_canonical, attachment_type)
		VALUES ($1, $2, $3, 'application/pdf', $4, true, 'original')`,
		antonioBirthCertID, "seed/antonio-rossi/birth-certificate.pdf", "birth-certificate.pdf", 1310720)

	// Antonio — Marriage certificate
	s.exec(`
		INSERT INTO file_attachments
			(document_id, storage_key, filename, content_type, size_bytes, is_canonical, attachment_type)
		VALUES ($1, $2, $3, 'application/pdf', $4, true, 'original')`,
		antonioMarriageCertID, "seed/antonio-rossi/marriage-certificate.pdf", "marriage-certificate.pdf", 1572864)

	// Antonio — Death certificate
	s.exec(`
		INSERT INTO file_attachments
			(document_id, storage_key, filename, content_type, size_bytes, is_canonical, attachment_type)
		VALUES ($1, $2, $3, 'application/pdf', $4, true, 'original')`,
		antonioDeathCertID, "seed/antonio-rossi/death-certificate.pdf", "death-certificate.pdf", 1048576)

	// Carlo — US birth certificate
	s.exec(`
		INSERT INTO file_attachments
			(document_id, storage_key, filename, content_type, size_bytes, is_canonical, attachment_type)
		VALUES ($1, $2, $3, 'application/pdf', $4, true, 'original')`,
		carloBirthCertID, "seed/carlo-rossi/birth-certificate.pdf", "birth-certificate.pdf", 1310720)

	// Carlo — Marriage certificate
	s.exec(`
		INSERT INTO file_attachments
			(document_id, storage_key, filename, content_type, size_bytes, is_canonical, attachment_type)
		VALUES ($1, $2, $3, 'application/pdf', $4, true, 'original')`,
		carloMarriageCertID, "seed/carlo-rossi/marriage-certificate.pdf", "marriage-certificate.pdf", 1572864)

	// Sofia — US birth certificate
	s.exec(`
		INSERT INTO file_attachments
			(document_id, storage_key, filename, content_type, size_bytes, is_canonical, attachment_type)
		VALUES ($1, $2, $3, 'application/pdf', $4, true, 'original')`,
		sofiaBirthCertID, "seed/sofia-martini/birth-certificate.pdf", "birth-certificate.pdf", 1310720)

	if s.err != nil {
		return fmt.Errorf("seed insert: %w", s.err)
	}

	return tx.Commit()
}
