/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Card, Title, Stack, Group, Badge, Text } from '@mantine/core';
import type { ClaimLineSummary } from '../../api/cases';
import { CLAIM_STATUS_COLOR, claimStatusLabel } from '../claim-lines/options';

const CLAIM_STATUSES = ['eligible', 'researching', 'not_yet_researched', 'paused', 'ineligible'] as const;

export default function ClaimLineSummaryCard({ summary }: { summary: ClaimLineSummary }) {
  return (
    <Card withBorder padding="lg" maw={400}>
      <Title order={4} mb="md">Claim Lines</Title>
      <Stack gap={6}>
        {CLAIM_STATUSES.map((s) =>
          summary[s] > 0 ? (
            <Group key={s} gap="xs">
              <Badge color={CLAIM_STATUS_COLOR[s]} variant="dot" size="sm">
                {claimStatusLabel(s)}
              </Badge>
              <Text size="sm">{summary[s]}</Text>
            </Group>
          ) : null,
        )}
        {summary.total === 0 && <Text size="sm" c="dimmed">No claim lines.</Text>}
      </Stack>
    </Card>
  );
}
