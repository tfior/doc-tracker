/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Link } from 'react-router-dom';
import { Badge, Table, Text, Button, Group } from '@mantine/core';
import type { Case } from '../../api/cases';
import { CASE_STATUS_COLOR, caseStatusLabel } from './options';

interface CaseTableProps {
  cases: Case[];
  onEdit: (c: Case) => void;
  onDelete: (c: Case) => void;
}

export default function CaseTable({ cases, onEdit, onDelete }: CaseTableProps) {
  return (
    <Table striped highlightOnHover withTableBorder>
      <Table.Thead>
        <Table.Tr>
          <Table.Th>Title</Table.Th>
          <Table.Th>Status</Table.Th>
          <Table.Th />
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {cases.length === 0 && (
          <Table.Tr>
            <Table.Td colSpan={3}>
              <Text c="dimmed">No cases yet.</Text>
            </Table.Td>
          </Table.Tr>
        )}
        {cases.map((c) => (
          <Table.Tr key={c.id}>
            <Table.Td>
              <Link to={`/cases/${c.id}`}>{c.title}</Link>
            </Table.Td>
            <Table.Td>
              <Badge color={CASE_STATUS_COLOR[c.status] ?? 'gray'}>
                {caseStatusLabel(c.status)}
              </Badge>
            </Table.Td>
            <Table.Td>
              <Group gap="xs" justify="flex-end">
                <Button size="xs" variant="subtle" onClick={() => onEdit(c)}>Edit</Button>
                <Button size="xs" variant="subtle" color="red" onClick={() => onDelete(c)}>Delete</Button>
              </Group>
            </Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  );
}
