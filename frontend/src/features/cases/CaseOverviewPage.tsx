import { useQuery } from '@tanstack/react-query';
import { useParams, Link } from 'react-router-dom';
import {
  Title, Text, Loader, Alert, Group, Badge, Stack,
  SimpleGrid, Card, RingProgress, ThemeIcon, List,
} from '@mantine/core';
import { getCase } from '../../api/cases';
import { listLifeEvents } from '../../api/lifeevents';

const BUCKET_COLOR: Record<string, string> = {
  not_started: 'gray',
  in_progress: 'yellow',
  complete: 'green',
};

const CLAIM_STATUS_COLOR: Record<string, string> = {
  active: 'blue',
  confirmed: 'green',
  suspended: 'orange',
  eliminated: 'red',
};

function formatEventType(type: string): string {
  return type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function CaseOverviewPage() {
  const { caseId } = useParams<{ caseId: string }>();

  const caseQuery = useQuery({
    queryKey: ['cases', caseId],
    queryFn: () => getCase(caseId!),
  });

  const eventsQuery = useQuery({
    queryKey: ['life-events', caseId],
    queryFn: () => listLifeEvents(caseId!),
  });

  if (caseQuery.isLoading) return <Loader />;
  if (caseQuery.isError) return <Alert color="red">Failed to load case.</Alert>;

  const detail = caseQuery.data!;
  const { claim_line_summary: cls, document_progress: dp } = detail;

  const totalDocs = dp.not_started + dp.in_progress + dp.complete;
  const completePct = totalDocs > 0 ? Math.round((dp.complete / totalDocs) * 100) : 0;

  const flaggedEvents = eventsQuery.data?.items.filter((e) => !e.has_documents) ?? [];

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="flex-start">
        <div>
          <Title order={2}>{detail.title}</Title>
          <Text c="dimmed" size="sm" mt={4}>Case Overview</Text>
        </div>
        <Badge size="lg" color={detail.status === 'active' ? 'green' : 'gray'}>
          {detail.status}
        </Badge>
      </Group>

      <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
        {/* Document Progress */}
        <Card withBorder padding="lg">
          <Title order={4} mb="md">Document Progress</Title>
          <Group align="center" gap="xl">
            <RingProgress
              size={100}
              thickness={10}
              roundCaps
              sections={[{ value: completePct, color: 'green' }]}
              label={
                <Text ta="center" fw={700} size="sm">{completePct}%</Text>
              }
            />
            <Stack gap={6}>
              {(['complete', 'in_progress', 'not_started'] as const).map((bucket) => (
                <Group key={bucket} gap="xs">
                  <Badge color={BUCKET_COLOR[bucket]} variant="dot" size="sm">
                    {bucket.replace(/_/g, ' ')}
                  </Badge>
                  <Text size="sm">{dp[bucket]}</Text>
                </Group>
              ))}
            </Stack>
          </Group>
        </Card>

        {/* Claim Line Summary */}
        <Card withBorder padding="lg">
          <Title order={4} mb="md">Claim Lines</Title>
          <Stack gap={6}>
            {(['confirmed', 'active', 'suspended', 'eliminated'] as const).map((status) => (
              cls[status] > 0 && (
                <Group key={status} gap="xs">
                  <Badge color={CLAIM_STATUS_COLOR[status]} variant="dot" size="sm">
                    {status}
                  </Badge>
                  <Text size="sm">{cls[status]}</Text>
                </Group>
              )
            ))}
            {cls.total === 0 && <Text size="sm" c="dimmed">No claim lines.</Text>}
          </Stack>
        </Card>
      </SimpleGrid>

      {/* Life Events Without Documents */}
      {!eventsQuery.isLoading && flaggedEvents.length > 0 && (
        <Card withBorder padding="lg">
          <Title order={4} mb="xs">Life Events Missing Documents</Title>
          <Text size="sm" c="dimmed" mb="sm">
            These events have no documents associated yet.
          </Text>
          <List spacing="xs" size="sm">
            {flaggedEvents.map((e) => (
              <List.Item key={e.id}>
                <Group gap="xs">
                  <ThemeIcon color="orange" size="xs" radius="xl" variant="filled">
                    <span />
                  </ThemeIcon>
                  <Text size="sm">
                    {formatEventType(e.event_type)}
                    {e.event_date ? ` — ${e.event_date}` : ''}
                  </Text>
                </Group>
              </List.Item>
            ))}
          </List>
        </Card>
      )}

      {/* Navigation */}
      <Group gap="md">
        <Link to={`/cases/${caseId}/people`}>People</Link>
        <Link to={`/cases/${caseId}/documents`}>Documents</Link>
      </Group>
    </Stack>
  );
}
