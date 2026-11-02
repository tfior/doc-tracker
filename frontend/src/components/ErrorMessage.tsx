/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Alert, Button, List, Text } from '@mantine/core';
import { errorMessage, fieldErrors, isVersionConflict } from '../lib/errors';

interface ErrorMessageProps {
  error: unknown;
  /** Heading for the message, such as what was being attempted. */
  title?: string;
  /** Fields the form shows its own errors beside; their messages are not repeated here. */
  shownFields?: string[];
  /** Offered when the record changed since it was read. */
  onReload?: () => Promise<void>;
}

/**
 * Shows why a request failed, in the server's own words. A version conflict
 * explains that someone else changed the record and offers to reload it.
 */
export default function ErrorMessage({ error, title, shownFields = [], onReload }: ErrorMessageProps) {
  const [reloading, setReloading] = useState(false);
  const [reloadError, setReloadError] = useState<unknown>(null);

  // A failed reload belongs to the conflict it was tried for, not to the next one.
  const [prevError, setPrevError] = useState(error);
  if (error !== prevError) {
    setPrevError(error);
    setReloadError(null);
  }

  if (!error) return null;

  if (isVersionConflict(error)) {
    async function reload() {
      setReloading(true);
      setReloadError(null);
      try {
        await onReload!();
      } catch (err) {
        setReloadError(err);
      } finally {
        setReloading(false);
      }
    }

    return (
      <Alert color="orange" title={title ?? 'Changed by someone else'}>
        <Text size="sm">{errorMessage(error)}</Text>
        {onReload && (
          <Button size="xs" mt="xs" variant="light" color="orange" loading={reloading} onClick={reload}>
            Reload
          </Button>
        )}
        {reloadError != null && <Text size="sm" mt="xs" c="red">{errorMessage(reloadError)}</Text>}
      </Alert>
    );
  }

  const message = errorMessage(error);
  const otherFields = Object.entries(fieldErrors(error))
    .filter(([field, text]) => !shownFields.includes(field) && text !== message);

  return (
    <Alert color="red" title={title}>
      <Text size="sm">{message}</Text>
      {otherFields.length > 0 && (
        <List size="sm" mt="xs">
          {otherFields.map(([field, message]) => (
            <List.Item key={field}>{field}: {message}</List.Item>
          ))}
        </List>
      )}
    </Alert>
  );
}
