import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Badge, Table, Title, Text, Loader, Alert } from '@mantine/core';
import { listCases, type Case } from '../../api/cases';

const STATUS_COLOR: Record<string, string> = {
  active: 'green',
  closed: 'gray',
  suspended: 'orange',
};

function statusLabel(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

export default function CaseListPage() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['cases'],
    queryFn: listCases,
  });

  if (isLoading) return <Loader />;
  if (isError) return <Alert color="red">Failed to load cases.</Alert>;

  const cases: Case[] = data?.items ?? [];

  return (
    <>
      <Title order={2} mb="md">Cases</Title>
      <Table striped highlightOnHover withTableBorder>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Title</Table.Th>
            <Table.Th>Status</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {cases.length === 0 && (
            <Table.Tr>
              <Table.Td colSpan={2}>
                <Text c="dimmed">No cases found.</Text>
              </Table.Td>
            </Table.Tr>
          )}
          {cases.map((c) => (
            <Table.Tr key={c.id}>
              <Table.Td>
                <Link to={`/cases/${c.id}`}>{c.title}</Link>
              </Table.Td>
              <Table.Td>
                <Badge color={STATUS_COLOR[c.status] ?? 'gray'}>
                  {statusLabel(c.status)}
                </Badge>
              </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </>
  );
}
