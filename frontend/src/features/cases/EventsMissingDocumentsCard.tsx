/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Card, Title, Text, List, Group, ThemeIcon } from '@mantine/core';
import type { LifeEvent } from '../../api/lifeevents';
import type { Person } from '../../api/people';
import { eventTypeLabel } from '../life-events/options';
import { eventPeople } from '../people/personName';
import { formatQualifiedDate } from '../../lib/dates';

interface EventsMissingDocumentsCardProps {
  events: LifeEvent[];
  people: Person[];
}

export default function EventsMissingDocumentsCard({ events, people }: EventsMissingDocumentsCardProps) {
  const flagged = events.filter((e) => !e.has_documents);
  if (flagged.length === 0) return null;

  return (
    <Card withBorder padding="lg">
      <Title order={4} mb="xs">Life Events Missing Documents</Title>
      <Text size="sm" c="dimmed" mb="sm">
        These events have no documents associated yet.
      </Text>
      <List spacing="xs" size="sm">
        {flagged.map((e) => (
          <List.Item key={e.id}>
            <Group gap="xs">
              <ThemeIcon color="orange" size="xs" radius="xl" variant="filled">
                <span />
              </ThemeIcon>
              <Text size="sm">
                {eventPeople(e, people)}: {eventTypeLabel(e.event_type)}
                {e.event_date ? ` — ${formatQualifiedDate(e.event_date)}` : ''}
              </Text>
            </Group>
          </List.Item>
        ))}
      </List>
    </Card>
  );
}
