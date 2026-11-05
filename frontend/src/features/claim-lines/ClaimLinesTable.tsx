/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Table, Text, Button, Group, Badge } from '@mantine/core';
import type { ClaimLine } from '../../api/claimlines';
import type { Person } from '../../api/people';
import { personName } from '../people/personName';
import { CLAIM_STATUS_COLOR, claimStatusLabel } from './options';

interface ClaimLinesTableProps {
  lines: ClaimLine[];
  people: Person[];
  onEdit: (line: ClaimLine) => void;
  onDelete: (line: ClaimLine) => void;
}

export default function ClaimLinesTable({ lines, people, onEdit, onDelete }: ClaimLinesTableProps) {
  return (
    <Table striped highlightOnHover withTableBorder>
      <Table.Thead>
        <Table.Tr>
          <Table.Th>Root Ancestor</Table.Th>
          <Table.Th>Status</Table.Th>
          <Table.Th>Notes</Table.Th>
          <Table.Th />
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {lines.length === 0 && (
          <Table.Tr>
            <Table.Td colSpan={4}>
              <Text c="dimmed">No claim lines yet.</Text>
            </Table.Td>
          </Table.Tr>
        )}
        {lines.map((line) => (
          <Table.Tr key={line.id} onClick={() => onEdit(line)} style={{ cursor: 'pointer' }}>
            <Table.Td>{personName(line.root_person_id, people)}</Table.Td>
            <Table.Td>
              <Badge color={CLAIM_STATUS_COLOR[line.status] ?? 'gray'}>
                {claimStatusLabel(line.status)}
              </Badge>
            </Table.Td>
            <Table.Td>
              <Text size="sm" c="dimmed" lineClamp={1}>{line.notes ?? '—'}</Text>
            </Table.Td>
            <Table.Td onClick={(e) => e.stopPropagation()}>
              <Group gap="xs" justify="flex-end">
                <Button size="xs" variant="subtle" color="red" onClick={() => onDelete(line)}>
                  Delete
                </Button>
              </Group>
            </Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  );
}
