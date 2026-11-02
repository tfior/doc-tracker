/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { Title, Text, Loader, Group, Badge, Stack, Button } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { caseQuery, lifeEventsQuery, peopleQuery } from '../../api/queries';
import ErrorMessage from '../../components/ErrorMessage';
import { CASE_STATUS_COLOR, caseStatusLabel } from './options';
import CaseFormModal from './CaseFormModal';
import ClaimLineSummaryCard from './ClaimLineSummaryCard';
import EventsMissingDocumentsCard from './EventsMissingDocumentsCard';

export default function CaseOverviewPage() {
  const { caseId } = useParams<{ caseId: string }>();
  const detailQuery = useQuery(caseQuery(caseId!));
  const eventsQuery = useQuery(lifeEventsQuery(caseId!));
  const people = useQuery(peopleQuery(caseId!)).data?.items ?? [];
  const [editOpened, editHandlers] = useDisclosure(false);

  if (detailQuery.isLoading) return <Loader />;
  if (detailQuery.error) return <ErrorMessage error={detailQuery.error} title="Could not load this case" />;

  const detail = detailQuery.data!;

  return (
    <>
      <Stack gap="lg">
        <Group justify="space-between" align="flex-start">
          <div>
            <Title order={2}>{detail.title}</Title>
            <Text c="dimmed" size="sm" mt={4}>Case Overview</Text>
          </div>
          <Group gap="xs">
            <Badge size="lg" color={CASE_STATUS_COLOR[detail.status] ?? 'gray'}>
              {caseStatusLabel(detail.status)}
            </Badge>
            <Button size="xs" variant="default" onClick={editHandlers.open}>Edit</Button>
          </Group>
        </Group>

        <ClaimLineSummaryCard summary={detail.claim_line_summary} />
        <ErrorMessage error={eventsQuery.error} title="Could not load life events" />
        {eventsQuery.data && <EventsMissingDocumentsCard events={eventsQuery.data.items} people={people} />}
      </Stack>

      <CaseFormModal record={detail} opened={editOpened} onClose={editHandlers.close} />
    </>
  );
}
