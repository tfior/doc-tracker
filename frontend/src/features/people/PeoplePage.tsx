import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { Title, Table, Text, Loader, Alert } from '@mantine/core';
import { listPeople, type Person } from '../../api/people';

function fullName(p: Person): string {
  return `${p.first_name} ${p.last_name}`;
}

export default function PeoplePage() {
  const { caseId } = useParams<{ caseId: string }>();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['people', caseId],
    queryFn: () => listPeople(caseId!),
  });

  if (isLoading) return <Loader />;
  if (isError) return <Alert color="red">Failed to load people.</Alert>;

  const people: Person[] = data?.items ?? [];

  return (
    <>
      <Title order={2} mb="md">People</Title>
      <Table striped highlightOnHover withTableBorder>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Name</Table.Th>
            <Table.Th>Birth Date</Table.Th>
            <Table.Th>Birth Place</Table.Th>
            <Table.Th>Death Date</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {people.length === 0 && (
            <Table.Tr>
              <Table.Td colSpan={4}>
                <Text c="dimmed">No people found.</Text>
              </Table.Td>
            </Table.Tr>
          )}
          {people.map((p) => (
            <Table.Tr key={p.id}>
              <Table.Td>{fullName(p)}</Table.Td>
              <Table.Td>{p.birth_date ?? '—'}</Table.Td>
              <Table.Td>{p.birth_place ?? '—'}</Table.Td>
              <Table.Td>{p.death_date ?? '—'}</Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </>
  );
}
