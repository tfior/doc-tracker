/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Component, type ErrorInfo, type ReactNode } from 'react';
import { Alert, Button, Text } from '@mantine/core';

interface ErrorBoundaryProps {
  children: ReactNode;
}

interface ErrorBoundaryState {
  failed: boolean;
}

/** Shows a message in place of a component that crashed, instead of a blank screen. */
export default class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { failed: false };

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { failed: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error(error, info.componentStack);
  }

  render() {
    if (!this.state.failed) return this.props.children;
    return (
      <Alert color="red" title="Something went wrong" m="md">
        <Text size="sm">This part of the page stopped working. Reloading the page usually fixes it.</Text>
        <Button size="xs" mt="xs" variant="light" color="red" onClick={() => window.location.reload()}>
          Reload page
        </Button>
      </Alert>
    );
  }
}
