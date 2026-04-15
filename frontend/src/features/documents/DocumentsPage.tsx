import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { Title, Table, Text, Badge, Loader, Alert } from '@mantine/core';
import { listDocuments, type Document } from '../../api/documents';

const BUCKET_COLOR: Record<string, string> = {
  not_started: 'gray',
  in_progress: 'yellow',
  complete: 'green',
};

export default function DocumentsPage() {
  const { caseId } = useParams<{ caseId: string }>();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['documents', caseId],
    queryFn: () => listDocuments(caseId!),
  });

  if (isLoading) return <Loader />;
  if (isError) return <Alert color="red">Failed to load documents.</Alert>;

  const documents: Document[] = data?.items ?? [];

  return (
    <>
      <Title order={2} mb="md">Documents</Title>
      <Table striped highlightOnHover withTableBorder>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Title</Table.Th>
            <Table.Th>Type</Table.Th>
            <Table.Th>Status</Table.Th>
            <Table.Th>Progress</Table.Th>
            <Table.Th>Verified</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {documents.length === 0 && (
            <Table.Tr>
              <Table.Td colSpan={5}>
                <Text c="dimmed">No documents found.</Text>
              </Table.Td>
            </Table.Tr>
          )}
          {documents.map((d) => (
            <Table.Tr key={d.id}>
              <Table.Td>{d.title}</Table.Td>
              <Table.Td>
                <Text size="sm" c="dimmed">
                  {d.document_type.replace(/_/g, ' ')}
                </Text>
              </Table.Td>
              <Table.Td>{d.status}</Table.Td>
              <Table.Td>
                <Badge color={BUCKET_COLOR[d.progress_bucket] ?? 'gray'} size="sm">
                  {d.progress_bucket.replace(/_/g, ' ')}
                </Badge>
              </Table.Td>
              <Table.Td>
                {d.is_verified ? (
                  <Badge color="green" size="sm">Verified</Badge>
                ) : (
                  <Badge color="gray" variant="outline" size="sm">Unverified</Badge>
                )}
              </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </>
  );
}
