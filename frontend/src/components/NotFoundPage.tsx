/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Link } from 'react-router-dom';
import { Title, Text, Anchor, Stack } from '@mantine/core';

export default function NotFoundPage() {
  return (
    <Stack gap="xs">
      <Title order={2}>Page not found</Title>
      <Text c="dimmed">There is no page at this address.</Text>
      <Anchor component={Link} to="/cases">Go to your cases</Anchor>
    </Stack>
  );
}
