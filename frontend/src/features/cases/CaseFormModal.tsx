/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Modal } from '@mantine/core';
import { createCase, updateCase, type Case } from '../../api/cases';
import { caseQuery } from '../../api/queries';
import { changedFields } from '../../lib/changes';
import CaseForm from './CaseForm';
import { caseFormValues, type CaseFormValues } from './caseFormValues';

interface CaseFormModalProps {
  /** The case to edit, or null to create one. */
  record: Case | null;
  opened: boolean;
  onClose: () => void;
  onCreated?: (c: Case) => void;
}

export default function CaseFormModal({ record, opened, onClose, onCreated }: CaseFormModalProps) {
  return (
    <Modal opened={opened} onClose={onClose} title={record ? 'Edit Case' : 'New Case'}>
      <CaseFormBody record={record} onClose={onClose} onCreated={onCreated} />
    </Modal>
  );
}

// The modal's content is unmounted when it closes, so each opening starts from
// the record as it was when opened, and saves against that version.
function CaseFormBody({ record, onClose, onCreated }: Omit<CaseFormModalProps, 'opened'>) {
  const queryClient = useQueryClient();
  const [base, setBase] = useState(record);

  const save = useMutation({
    mutationFn: async (values: CaseFormValues) => {
      if (!base) return createCase(values.title);
      const input = changedFields(caseFormValues(base), values);
      if (Object.keys(input).length === 0) return base;
      return updateCase(base.id, base.version, input);
    },
    onSuccess: (saved) => {
      queryClient.invalidateQueries({ queryKey: ['cases'] });
      if (!base) onCreated?.(saved);
      onClose();
    },
  });

  async function reload() {
    const fresh = await queryClient.fetchQuery({ ...caseQuery(base!.id), staleTime: 0 });
    save.reset();
    setBase(fresh);
  }

  return (
    <CaseForm
      key={base?.version ?? 'new'}
      initial={caseFormValues(base)}
      isNew={!base}
      saving={save.isPending}
      error={save.error}
      onSubmit={(values) => save.mutate(values)}
      onCancel={onClose}
      onReload={reload}
    />
  );
}
