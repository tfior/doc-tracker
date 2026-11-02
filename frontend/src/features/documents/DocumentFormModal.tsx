/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@mantine/core';
import { createDocument, updateDocument, type Document } from '../../api/documents';
import {
  documentsQuery, documentStatusesQuery, invalidateCase, lifeEventsQuery, peopleQuery,
} from '../../api/queries';
import { changedFields } from '../../lib/changes';
import DocumentForm from './DocumentForm';
import { documentFormValues, documentInput, type DocumentFormValues } from './documentFormValues';

interface DocumentFormModalProps {
  caseId: string;
  /** The document to edit, or null to add one. */
  record: Document | null;
  opened: boolean;
  onClose: () => void;
  /** When set, the person field is hidden and a new document belongs to this person. */
  lockedPersonId?: string;
  /** When set, the life event field is hidden and a new document belongs to this event. */
  lockedLifeEventId?: string;
}

export default function DocumentFormModal(props: DocumentFormModalProps) {
  const { record, opened, onClose } = props;
  return (
    <Modal opened={opened} onClose={onClose} title={record ? `Edit — ${record.title}` : 'Add Document'} size="xl">
      <DocumentFormBody {...props} />
    </Modal>
  );
}

// The modal's content is unmounted when it closes, so each opening starts from
// the record as it was when opened, and saves against that version.
function DocumentFormBody({ caseId, record, onClose, lockedPersonId, lockedLifeEventId }: DocumentFormModalProps) {
  const queryClient = useQueryClient();
  const people = useQuery(peopleQuery(caseId)).data?.items ?? [];
  const lifeEvents = useQuery(lifeEventsQuery(caseId)).data?.items ?? [];
  const statuses = useQuery(documentStatusesQuery()).data ?? [];
  const [base, setBase] = useState(record);
  const defaults = { personId: lockedPersonId, lifeEventId: lockedLifeEventId };

  const save = useMutation({
    mutationFn: async (values: DocumentFormValues) => {
      if (!base) return createDocument(caseId, documentInput(values, true));
      const input = changedFields(documentInput(documentFormValues(base), false), documentInput(values, false));
      if (Object.keys(input).length === 0) return base;
      return updateDocument(caseId, base.id, base.version, input);
    },
    onSuccess: () => {
      invalidateCase(queryClient, caseId);
      onClose();
    },
  });

  async function reload() {
    const { items } = await queryClient.fetchQuery({ ...documentsQuery(caseId), staleTime: 0 });
    const fresh = items.find((d) => d.id === base!.id);
    save.reset();
    if (fresh) setBase(fresh);
    else onClose();
  }

  return (
    <DocumentForm
      key={base?.version ?? 'new'}
      initial={documentFormValues(base, defaults)}
      isNew={!base}
      people={people}
      lifeEvents={lifeEvents}
      statuses={statuses}
      hidePerson={!!lockedPersonId}
      hideLifeEvent={!!lockedLifeEventId}
      saving={save.isPending}
      error={save.error}
      onSubmit={(values) => save.mutate(values)}
      onCancel={onClose}
      onReload={reload}
    />
  );
}
