export async function fetchBootstrap() {
  const response = await fetch('/api/bootstrap');
  if (!response.ok) {
    throw new Error(`Bootstrap failed with status ${response.status}`);
  }
  return response.json();
}

export async function postInteraction(command) {
  const response = await fetch('/api/interactions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(command)
  });

  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(body.error || `Interaction failed with status ${response.status}`);
  }

  return body;
}