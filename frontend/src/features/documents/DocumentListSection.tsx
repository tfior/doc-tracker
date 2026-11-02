/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Divider, Group, Text, Button, Stack } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import type { Document } from '../../api/documents';
import { docTypeLabel, phaseStatus } from './options';
import DocumentFormModal from './DocumentFormModal';
import DocumentDeleteModal from './DocumentDeleteModal';

interface DocumentListSectionProps {
  caseId: string;
  documents: Document[];
  emptyText: string;
  /** New documents belong to this person. */
  personId: string;
  /** New documents belong to this event; its documents cannot be moved from here. */
  lifeEventId?: string;
}

/** A person's or life event's documents, managed from inside its modal. */
export default function DocumentListSection({ caseId, documents, emptyText, personId, lifeEventId }: DocumentListSectionProps) {
  const [formOpened, formHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editing, setEditing] = useState<Document | null>(null);
  const [deleting, setDeleting] = useState<Document | null>(null);

  function openForm(doc: Document | null) {
    setEditing(doc);
    formHandlers.open();
  }

  function openDelete(doc: Document) {
    setDeleting(doc);
    deleteHandlers.open();
  }

  return (
    <>
      <Divider mt="xs" />
      <Group justify="space-between" align="center">
        <Text fw={500} size="sm">Documents</Text>
        <Button size="xs" variant="light" onClick={() => openForm(null)}>Add</Button>
      </Group>
      {documents.length === 0 && <Text size="sm" c="dimmed">{emptyText}</Text>}
      <Stack gap="xs">
        {documents.map((doc) => (
          <Group key={doc.id} justify="space-between" wrap="nowrap"
            style={{ borderBottom: '1px solid var(--mantine-color-gray-2)', paddingBottom: 8 }}
          >
            <div>
              <Text size="sm" fw={500}>{doc.title}</Text>
              <Text size="xs" c="dimmed">
                {docTypeLabel(doc.document_type)} · {phaseStatus(doc, 'official_copy').label}
              </Text>
            </div>
            <Group gap="xs" wrap="nowrap">
              <Button size="xs" variant="subtle" onClick={() => openForm(doc)}>Edit</Button>
              <Button size="xs" variant="subtle" color="red" onClick={() => openDelete(doc)}>Delete</Button>
            </Group>
          </Group>
        ))}
      </Stack>

      <DocumentFormModal
        caseId={caseId}
        record={editing}
        opened={formOpened}
        onClose={formHandlers.close}
        lockedPersonId={personId}
        lockedLifeEventId={lifeEventId}
      />
      <DocumentDeleteModal caseId={caseId} record={deleting} opened={deleteOpened} onClose={deleteHandlers.close} />
    </>
  );
}
