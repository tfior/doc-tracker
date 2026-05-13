import { Routes, Route, Navigate, Outlet, useNavigate } from 'react-router-dom';
import { AppShell, NavLink, Title, Group, Anchor, Button, Loader, Center } from '@mantine/core';
import { Link, useParams } from 'react-router-dom';
import { AuthProvider, useAuth } from './features/auth/AuthProvider';
import LoginPage from './features/auth/LoginPage';
import CaseListPage from './features/cases/CaseListPage';
import CaseOverviewPage from './features/cases/CaseOverviewPage';
import PeoplePage from './features/people/PeoplePage';
import DocumentsPage from './features/documents/DocumentsPage';
import LifeEventsPage from './features/life-events/LifeEventsPage';
import ClaimLinesPage from './features/claim-lines/ClaimLinesPage';
import TrashPage from './features/trash/TrashPage';
import GlobalTrashPage from './features/trash/GlobalTrashPage';

function CaseNav() {
  const { caseId } = useParams<{ caseId: string }>();
  return (
    <Group gap="xs" mb="md">
      <Anchor component={Link} to={`/cases/${caseId}`} size="sm">Overview</Anchor>
      <Anchor component={Link} to={`/cases/${caseId}/people`} size="sm">People</Anchor>
      <Anchor component={Link} to={`/cases/${caseId}/life-events`} size="sm">Life Events</Anchor>
      <Anchor component={Link} to={`/cases/${caseId}/documents`} size="sm">Documents</Anchor>
      <Anchor component={Link} to={`/cases/${caseId}/claim-lines`} size="sm">Claim Lines</Anchor>

      <Anchor component={Link} to={`/cases/${caseId}/trash`} size="sm">Trash</Anchor>
    </Group>
  );
}

function RequireAuth() {
  const { authenticated, checking } = useAuth();
  if (checking) return <Center h="100vh"><Loader /></Center>;
  if (!authenticated) return <Navigate to="/login" replace />;
  return <Outlet />;
}

function AppLayout() {
  const { logout } = useAuth();
  const navigate = useNavigate();

  async function handleLogout() {
    await logout();
    navigate('/login', { replace: true });
  }

  return (
    <AppShell
      header={{ height: 56 }}
      navbar={{ width: 220, breakpoint: 'sm' }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Link to="/cases" style={{ textDecoration: 'none', color: 'inherit' }}>
            <Title order={4}>Doc Tracker</Title>
          </Link>
          <Button variant="subtle" size="xs" onClick={handleLogout}>
            Sign out
          </Button>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="xs">
        <NavLink component={Link} to="/cases" label="Cases" />
        <NavLink component={Link} to="/trash" label="Trash" />
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<RequireAuth />}>
          <Route element={<AppLayout />}>
            <Route path="/" element={<Navigate to="/cases" replace />} />
            <Route path="/trash" element={<GlobalTrashPage />} />
            <Route path="/cases" element={<CaseListPage />} />
            <Route path="/cases/:caseId" element={<><CaseNav /><CaseOverviewPage /></>} />
            <Route
              path="/cases/:caseId/people"
              element={<><CaseNav /><PeoplePage /></>}
            />
            <Route
              path="/cases/:caseId/life-events"
              element={<><CaseNav /><LifeEventsPage /></>}
            />
            <Route
              path="/cases/:caseId/documents"
              element={<><CaseNav /><DocumentsPage /></>}
            />
            <Route
              path="/cases/:caseId/claim-lines"
              element={<><CaseNav /><ClaimLinesPage /></>}
            />
            <Route
              path="/cases/:caseId/trash"
              element={<><CaseNav /><TrashPage /></>}
            />
          </Route>
        </Route>
      </Routes>
    </AuthProvider>
  );
}
