/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Table, Text, Button, Group, Badge } from '@mantine/core';
import type { LifeEvent } from '../../api/lifeevents';
import type { Person } from '../../api/people';
import { eventPeople } from '../people/personName';
import { eventTypeLabel } from './options';
import { formatQualifiedDate } from '../../lib/dates';

interface LifeEventsTableProps {
  lifeEvents: LifeEvent[];
  people: Person[];
  onEdit: (le: LifeEvent) => void;
  onDelete: (le: LifeEvent) => void;
}

export default function LifeEventsTable({ lifeEvents, people, onEdit, onDelete }: LifeEventsTableProps) {
  return (
    <Table striped highlightOnHover withTableBorder>
      <Table.Thead>
        <Table.Tr>
          <Table.Th>Person</Table.Th>
          <Table.Th>Event</Table.Th>
          <Table.Th>Date</Table.Th>
          <Table.Th>Place</Table.Th>
          <Table.Th>Docs</Table.Th>
          <Table.Th />
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {lifeEvents.length === 0 && (
          <Table.Tr>
            <Table.Td colSpan={6}>
              <Text c="dimmed">No life events yet.</Text>
            </Table.Td>
          </Table.Tr>
        )}
        {lifeEvents.map((e) => (
          <Table.Tr key={e.id} onClick={() => onEdit(e)} style={{ cursor: 'pointer' }}>
            <Table.Td>{eventPeople(e, people)}</Table.Td>
            <Table.Td>{eventTypeLabel(e.event_type)}</Table.Td>
            <Table.Td>{e.event_date ? formatQualifiedDate(e.event_date) : '—'}</Table.Td>
            <Table.Td>{e.event_place ?? '—'}</Table.Td>
            <Table.Td>
              {!e.has_documents && (
                <Badge color="orange" variant="dot" size="sm">No docs</Badge>
              )}
            </Table.Td>
            <Table.Td>
              <Group gap="xs" justify="flex-end" onClick={(ev) => ev.stopPropagation()}>
                <Button size="xs" variant="subtle" color="red" onClick={() => onDelete(e)}>
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
