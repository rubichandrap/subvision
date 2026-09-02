// Thin loader for the client's environment contract (documented in the
// README): the server URL is required and never silently defaulted — a
// missing variable fails with a clear message naming it.

const serverUrl = process.env.NEXT_PUBLIC_SERVER_URL;

if (!serverUrl) {
  throw new Error(
    'Missing required environment variable: NEXT_PUBLIC_SERVER_URL'
  );
}

export const env = {
  serverUrl,
};
