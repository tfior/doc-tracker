/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';

interface VersionedRecord {
  id: string;
  version: number;
}

interface VersionedDeleteOptions<T extends VersionedRecord> {
  /** The record as it was when the user chose to delete it. */
  record: T | null;
  opened: boolean;
  onClose: () => void;
  remove: (record: T) => Promise<void>;
  /** Fetches the latest records, for a reload after a version conflict. */
  load: () => Promise<T[]>;
  onDeleted: () => void;
}

/**
 * The state behind a delete confirmation. The delete is made against the
 * version the user saw when the dialog opened, so a change made elsewhere
 * since then is refused as a conflict rather than deleted unseen. Only Reload
 * moves it to a newer version.
 */
export function useVersionedDelete<T extends VersionedRecord>({
  record, opened, onClose, remove, load, onDeleted,
}: VersionedDeleteOptions<T>) {
  const [current, setCurrent] = useState(record);

  // Start again from the chosen record each time the dialog opens.
  const openedFor = opened ? record : null;
  const [prevOpenedFor, setPrevOpenedFor] = useState(openedFor);
  if (openedFor !== prevOpenedFor) {
    setPrevOpenedFor(openedFor);
    if (openedFor) setCurrent(openedFor);
  }

  const del = useMutation({
    mutationFn: remove,
    onSuccess: () => {
      onDeleted();
      close();
    },
  });

  function close() {
    del.reset();
    onClose();
  }

  async function reload() {
    const fresh = (await load()).find((r) => r.id === current?.id);
    del.reset();
    if (fresh) setCurrent(fresh);
    else close();
  }

  return {
    current,
    confirm: () => current && del.mutate(current),
    close,
    reload,
    pending: del.isPending,
    error: del.error,
  };
}
