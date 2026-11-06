/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Card, Title, Stack, Group, Badge, Text, Progress as Bar } from '@mantine/core';
import type { Progress } from '../../api/cases';
import { READINESS_COLOR, READINESS_LABELS } from '../documents/readiness';
import { progressSummary } from './progress';

const STATUSES = ['ready', 'in_progress', 'not_started'] as const;

/** How many of the case's documents for submission are ready. */
export default function DocumentProgressCard({ progress }: { progress: Progress }) {
  const { total, text } = progressSummary(progress);
  return (
    <Card withBorder padding="lg" maw={400}>
      <Title order={4} mb="md">Documents</Title>
      {total === 0 ? (
        <Text size="sm" c="dimmed">{text}.</Text>
      ) : (
        <Stack gap={6}>
          <Text size="sm" fw={500}>{text}</Text>
          <Bar value={(progress.ready / total) * 100} color={READINESS_COLOR.ready} size="sm" />
          {STATUSES.map((s) => progress[s] > 0 && (
            <Group key={s} gap="xs">
              <Badge color={READINESS_COLOR[s]} variant="dot" size="sm">{READINESS_LABELS[s]}</Badge>
              <Text size="sm">{progress[s]}</Text>
            </Group>
          ))}
        </Stack>
      )}
    </Card>
  );
}
