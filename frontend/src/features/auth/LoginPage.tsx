/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useLocation, useNavigate, type Location } from 'react-router-dom';
import {
  Center,
  Paper,
  Title,
  TextInput,
  PasswordInput,
  Button,
  Alert,
  Stack,
} from '@mantine/core';
import { useAuth } from './authContext';
import { errorMessage } from '../../lib/errors';

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  // Where the user was heading when they were sent here, if anywhere.
  const from = (useLocation().state as { from?: Location } | null)?.from;

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login(email, password);
      navigate(from ? `${from.pathname}${from.search}${from.hash}` : '/cases', { replace: true });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Center h="100vh">
      <Paper w={360} p="xl" withBorder shadow="sm">
        <Title order={3} mb="lg">Doc Tracker</Title>
        <form onSubmit={handleSubmit}>
          <Stack>
            {error && <Alert color="red">{error}</Alert>}
            <TextInput
              label="Email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.currentTarget.value)}
              required
              autoFocus
            />
            <PasswordInput
              label="Password"
              value={password}
              onChange={(e) => setPassword(e.currentTarget.value)}
              required
            />
            <Button type="submit" loading={loading} fullWidth mt="xs">
              Sign in
            </Button>
          </Stack>
        </form>
      </Paper>
    </Center>
  );
}
