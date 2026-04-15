import { Routes, Route, Navigate } from 'react-router-dom';
import { AppShell, NavLink, Title, Group, Anchor } from '@mantine/core';
import { Link, useParams } from 'react-router-dom';
import CaseListPage from './features/cases/CaseListPage';
import CaseOverviewPage from './features/cases/CaseOverviewPage';
import PeoplePage from './features/people/PeoplePage';
import DocumentsPage from './features/documents/DocumentsPage';

function CaseNav() {
  const { caseId } = useParams<{ caseId: string }>();
  return (
    <Group gap="xs" mb="md">
      <Anchor component={Link} to={`/cases/${caseId}`} size="sm">Overview</Anchor>
      <Anchor component={Link} to={`/cases/${caseId}/people`} size="sm">People</Anchor>
      <Anchor component={Link} to={`/cases/${caseId}/documents`} size="sm">Documents</Anchor>
    </Group>
  );
}

export default function App() {
  return (
    <AppShell
      header={{ height: 56 }}
      navbar={{ width: 220, breakpoint: 'sm' }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md">
          <Link to="/cases" style={{ textDecoration: 'none', color: 'inherit' }}>
            <Title order={4}>Doc Tracker</Title>
          </Link>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="xs">
        <NavLink component={Link} to="/cases" label="Cases" />
      </AppShell.Navbar>

      <AppShell.Main>
        <Routes>
          <Route path="/" element={<Navigate to="/cases" replace />} />
          <Route path="/cases" element={<CaseListPage />} />
          <Route path="/cases/:caseId" element={<CaseOverviewPage />} />
          <Route
            path="/cases/:caseId/people"
            element={<><CaseNav /><PeoplePage /></>}
          />
          <Route
            path="/cases/:caseId/documents"
            element={<><CaseNav /><DocumentsPage /></>}
          />
        </Routes>
      </AppShell.Main>
    </AppShell>
  );
}
