/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@mantine/core';
import { createClaimLine, updateClaimLine, type ClaimLine } from '../../api/claimlines';
import { claimLinesQuery, invalidateCase, peopleQuery } from '../../api/queries';
import { changedFields } from '../../lib/changes';
import ClaimLineForm from './ClaimLineForm';
import { claimLineFormValues, type ClaimLineFormValues } from './claimLineFormValues';

interface ClaimLineFormModalProps {
  caseId: string;
  /** The claim line to edit, or null to add one. */
  record: ClaimLine | null;
  opened: boolean;
  onClose: () => void;
}

export default function ClaimLineFormModal(props: ClaimLineFormModalProps) {
  const { record, opened, onClose } = props;
  return (
    <Modal opened={opened} onClose={onClose} title={record ? 'Edit Claim Line' : 'Add Claim Line'}>
      <ClaimLineFormBody {...props} />
    </Modal>
  );
}

// The modal's content is unmounted when it closes, so each opening starts from
// the record as it was when opened, and saves against that version.
function ClaimLineFormBody({ caseId, record, onClose }: ClaimLineFormModalProps) {
  const queryClient = useQueryClient();
  const people = useQuery(peopleQuery(caseId)).data?.items ?? [];
  const [base, setBase] = useState(record);

  const save = useMutation({
    mutationFn: async (values: ClaimLineFormValues) => {
      const notes = values.notes.trim() || null;
      if (!base) return createClaimLine(caseId, { root_person_id: values.root_person_id, status: values.status, notes });
      const before = claimLineFormValues(base);
      const input = changedFields({ status: before.status, notes: before.notes || null }, { status: values.status, notes });
      if (Object.keys(input).length === 0) return base;
      return updateClaimLine(caseId, base.id, base.version, input);
    },
    onSuccess: () => {
      invalidateCase(queryClient, caseId);
      onClose();
    },
  });

  async function reload() {
    const { items } = await queryClient.fetchQuery({ ...claimLinesQuery(caseId), staleTime: 0 });
    const fresh = items.find((l) => l.id === base!.id);
    save.reset();
    if (fresh) setBase(fresh);
    else onClose();
  }

  return (
    <ClaimLineForm
      key={base?.version ?? 'new'}
      initial={claimLineFormValues(base)}
      isNew={!base}
      people={people}
      saving={save.isPending}
      error={save.error}
      onSubmit={(values) => save.mutate(values)}
      onCancel={onClose}
      onReload={reload}
    />
  );
}
