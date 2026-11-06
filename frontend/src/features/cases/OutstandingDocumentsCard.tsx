/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Card, Title, Stack, Group, Text, Table, SegmentedControl } from '@mantine/core';
import type { Document } from '../../api/documents';
import type { Person } from '../../api/people';
import { formatDays } from '../../lib/dates';
import { personName } from '../people/personName';
import { STAGES, STAGE_LABELS } from '../documents/readiness';

type GroupBy = 'stage' | 'person';
type SortBy = 'longest' | 'shortest';

interface OutstandingGroup {
  key: string;
  label: string;
  docs: Document[];
}

/**
 * Every document for submission that is not ready, under the stage it is
 * waiting on or under its person. The server decides which documents are
 * outstanding and what each waits on; this only groups and sorts them.
 */
export default function OutstandingDocumentsCard({ documents, people }: { documents: Document[]; people: Person[] }) {
  const [groupBy, setGroupBy] = useState<GroupBy>('stage');
  const [sortBy, setSortBy] = useState<SortBy>('longest');

  const outstanding = documents
    .filter((d) => d.readiness?.waiting_on)
    .sort((a, b) => {
      const diff = Date.parse(a.readiness!.waiting_since!) - Date.parse(b.readiness!.waiting_since!);
      return sortBy === 'longest' ? diff : -diff;
    });

  const groups: OutstandingGroup[] = groupBy === 'stage'
    ? STAGES.map((s) => ({ key: s, label: STAGE_LABELS[s], docs: outstanding.filter((d) => d.readiness!.waiting_on === s) }))
    : [...new Set(outstanding.map((d) => d.person_id))]
      .map((id) => ({ key: id, label: personName(id, people), docs: outstanding.filter((d) => d.person_id === id) }))
      .sort((a, b) => a.label.localeCompare(b.label));

  return (
    <Card withBorder padding="lg">
      <Group justify="space-between" mb="md">
        <Title order={4}>Outstanding</Title>
        {outstanding.length > 0 && (
          <Group gap="xs">
            <SegmentedControl size="xs" value={groupBy} onChange={(v) => setGroupBy(v as GroupBy)}
              data={[{ value: 'stage', label: 'By stage' }, { value: 'person', label: 'By person' }]} />
            <SegmentedControl size="xs" value={sortBy} onChange={(v) => setSortBy(v as SortBy)}
              data={[{ value: 'longest', label: 'Longest waiting' }, { value: 'shortest', label: 'Most recent' }]} />
          </Group>
        )}
      </Group>
      {outstanding.length === 0 ? (
        <Text size="sm" c="dimmed">Nothing outstanding.</Text>
      ) : (
        <Stack gap="md">
          {groups.filter((g) => g.docs.length > 0).map((g) => (
            <div key={g.key}>
              <Text size="sm" fw={600} mb={4}>{g.label} ({g.docs.length})</Text>
              <Table withTableBorder style={{ tableLayout: 'fixed' }}>
                <Table.Tbody>
                  {g.docs.map((d) => (
                    <Table.Tr key={d.id}>
                      <Table.Td style={{ width: '40%' }}><Text size="sm" truncate>{d.title}</Text></Table.Td>
                      <Table.Td style={{ width: '30%' }}>
                        <Text size="sm" truncate>
                          {groupBy === 'stage' ? personName(d.person_id, people) : STAGE_LABELS[d.readiness!.waiting_on!]}
                        </Text>
                      </Table.Td>
                      <Table.Td style={{ width: '30%' }}>
                        <Text size="sm" c="dimmed">Waiting {formatDays(d.readiness!.waiting_days!)}</Text>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </div>
          ))}
        </Stack>
      )}
    </Card>
  );
}
