/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Table, Text, Button, Group, Badge } from '@mantine/core';
import type { Person } from '../../api/people';
import { fullName } from './personName';
import { formatQualifiedDate } from '../../lib/dates';

interface PeopleTableProps {
  people: Person[];
  onEdit: (p: Person) => void;
  onDelete: (p: Person) => void;
}

export default function PeopleTable({ people, onEdit, onDelete }: PeopleTableProps) {
  return (
    <Table striped highlightOnHover withTableBorder>
      <Table.Thead>
        <Table.Tr>
          <Table.Th>Name</Table.Th>
          <Table.Th>Birth Date</Table.Th>
          <Table.Th>Birth Place</Table.Th>
          <Table.Th>Death Date</Table.Th>
          <Table.Th>Parents</Table.Th>
          <Table.Th />
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {people.length === 0 && (
          <Table.Tr>
            <Table.Td colSpan={6}><Text c="dimmed">No people yet.</Text></Table.Td>
          </Table.Tr>
        )}
        {people.map((p) => {
          const parents = people.filter((other) => p.parent_ids.includes(other.id));
          return (
            <Table.Tr key={p.id} onClick={() => onEdit(p)} style={{ cursor: 'pointer' }}>
              <Table.Td>{fullName(p)}</Table.Td>
              <Table.Td>{p.birth_date ? formatQualifiedDate(p.birth_date) : '—'}</Table.Td>
              <Table.Td>{p.birth_place ?? '—'}</Table.Td>
              <Table.Td>{p.death_date ? formatQualifiedDate(p.death_date) : '—'}</Table.Td>
              <Table.Td>
                {parents.length === 0 ? (
                  <Text size="sm" c="dimmed">—</Text>
                ) : (
                  <Group gap={4}>
                    {parents.map((parent) => (
                      <Badge key={parent.id} variant="outline" size="sm">{fullName(parent)}</Badge>
                    ))}
                  </Group>
                )}
              </Table.Td>
              <Table.Td>
                <Group gap="xs" justify="flex-end" onClick={(e) => e.stopPropagation()}>
                  <Button size="xs" variant="subtle" color="red" onClick={() => onDelete(p)}>Delete</Button>
                </Group>
              </Table.Td>
            </Table.Tr>
          );
        })}
      </Table.Tbody>
    </Table>
  );
}
