/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Text, Button, Select, Badge, Stack, Tooltip } from '@mantine/core';
import { updateDocument, type Document, type DocumentStatus } from '../../api/documents';
import type { Items } from '../../api/client';
import type { Person } from '../../api/people';
import { documentsQuery, invalidateCase } from '../../api/queries';
import ErrorMessage from '../../components/ErrorMessage';
import { personName } from '../people/personName';
import { formatAge, formatDays } from '../../lib/dates';
import { READINESS_COLOR, READINESS_LABELS, daysInStatus, outsidePacketLabel, reasonLabel } from './readiness';
import { PHASES, PHASE_LABELS, docTypeLabel, phaseStatus, phaseStatusField, phaseStatusOptions, type Phase } from './options';

interface DocumentsTableProps {
  caseId: string;
  documents: Document[];
  people: Person[];
  statuses: DocumentStatus[];
  onEdit: (doc: Document) => void;
  onDelete: (doc: Document) => void;
}

interface StatusChange {
  doc: Document;
  phase: Phase;
  statusId: string;
}

export default function DocumentsTable({ caseId, documents, people, statuses, onEdit, onDelete }: DocumentsTableProps) {
  const queryClient = useQueryClient();

  const statusChange = useMutation({
    mutationFn: ({ doc, phase, statusId }: StatusChange) =>
      updateDocument(caseId, doc.id, doc.version, { [phaseStatusField(phase)]: statusId }),
    onSuccess: (saved) => {
      // Put the new version in place at once, so a second change to the same
      // document does not go out with the old one before the list reloads.
      queryClient.setQueryData(documentsQuery(caseId).queryKey, (old: Items<Document> | undefined) =>
        old && { ...old, items: old.items.map((d) => (d.id === saved.id ? saved : d)) },
      );
      invalidateCase(queryClient, caseId);
    },
  });

  async function reload() {
    await queryClient.refetchQueries({ queryKey: documentsQuery(caseId).queryKey });
    statusChange.reset();
  }

  const failed = statusChange.variables;

  return (
    <Stack>
      <ErrorMessage
        error={statusChange.error}
        title={failed && `Could not change the ${PHASE_LABELS[failed.phase].toLowerCase()} status of “${failed.doc.title}”`}
        onReload={reload}
      />
      <Table striped withTableBorder style={{ tableLayout: 'fixed' }}>
        <Table.Thead>
          <Table.Tr>
            <Table.Th style={{ width: '14%' }}>Title</Table.Th>
            <Table.Th style={{ width: '10%' }}>Person</Table.Th>
            <Table.Th style={{ width: '10%' }}>Type</Table.Th>
            {PHASES.map((phase) => (
              <Table.Th key={phase} style={{ width: '11%' }}>{PHASE_LABELS[phase]}</Table.Th>
            ))}
            <Table.Th style={{ width: '9%' }}>Readiness</Table.Th>
            <Table.Th style={{ width: '6%' }}>Checked</Table.Th>
            <Table.Th style={{ width: '6%' }} />
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {documents.length === 0 && (
            <Table.Tr>
              <Table.Td colSpan={10}><Text c="dimmed">No documents yet.</Text></Table.Td>
            </Table.Tr>
          )}
          {documents.map((d) => {
            const saving = statusChange.isPending && statusChange.variables?.doc.id === d.id;
            return (
              <Table.Tr key={d.id} onClick={() => onEdit(d)} style={{ cursor: 'pointer' }}>
                <Table.Td><Text size="sm" truncate>{d.title}</Text></Table.Td>
                <Table.Td><Text size="sm" truncate>{personName(d.person_id, people)}</Text></Table.Td>
                <Table.Td>
                  <Text size="xs" c="dimmed">{docTypeLabel(d.document_type)}</Text>
                  {d.copy_age_days !== null && (
                    <Text size="xs" c="dimmed">Copy {formatAge(d.copy_age_days)} old</Text>
                  )}
                </Table.Td>
                {PHASES.map((phase) => (
                  <Table.Td key={phase} onClick={(e) => e.stopPropagation()}>
                    <Select
                      size="xs"
                      aria-label={`${PHASE_LABELS[phase]} status of ${d.title}`}
                      data={phaseStatusOptions(statuses, phase)}
                      value={phaseStatus(d, phase).id}
                      disabled={saving}
                      onChange={(v) => v && statusChange.mutate({ doc: d, phase, statusId: v })}
                      styles={{ input: { minHeight: 'unset', fontSize: '11px' } }}
                    />
                    <Text size="xs" c="dimmed" mt={2}>{formatDays(daysInStatus(d, phase))}</Text>
                  </Table.Td>
                ))}
                <Table.Td><ReadinessBadge doc={d} /></Table.Td>
                <Table.Td>
                  {d.details_checked
                    ? <Badge color="green" size="xs">Yes</Badge>
                    : <Badge color="gray" variant="outline" size="xs">No</Badge>}
                </Table.Td>
                <Table.Td onClick={(e) => e.stopPropagation()}>
                  <Button size="xs" variant="subtle" color="red" onClick={() => onDelete(d)}>
                    Delete
                  </Button>
                </Table.Td>
              </Table.Tr>
            );
          })}
        </Table.Tbody>
      </Table>
    </Stack>
  );
}

// A document's readiness, with the reasons it is not ready on hover; or why it
// has none.
function ReadinessBadge({ doc }: { doc: Document }) {
  const r = doc.readiness;
  if (!r) return <Badge color="gray" variant="outline" size="xs">{outsidePacketLabel(doc)}</Badge>;
  const badge = <Badge color={READINESS_COLOR[r.status]} size="xs">{READINESS_LABELS[r.status]}</Badge>;
  if (r.reasons.length === 0) return badge;
  return (
    <Tooltip multiline w={240} label={<Stack gap={2}>{r.reasons.map((c) => <Text key={c} size="xs">{reasonLabel(c)}</Text>)}</Stack>}>
      {badge}
    </Tooltip>
  );
}
