// Client REST API for M2 Server Online (ADR-003)

const SERVER_BASE =
  import.meta.env.VITE_SERVER_URL ||
  (window.location.protocol === 'https:' ? 'https://' : 'http://') +
    (window.location.hostname || 'localhost') +
    ':8080';

export interface GuestAuthResponse {
  token: string;
  userId: string;
  displayName: string;
}

export interface MatchSummary {
  id: string;
  status: string;
  seed: number;
  contentHash: string;
  playerIds: string[];
  turnTimeoutSec: number;
  createdAt: string;
}

export interface CreateMatchRequest {
  matchId?: string;
  seed?: number;
  players: { id: string; name: string; character: string }[];
}

export const api = {
  async authGuest(displayName: string): Promise<GuestAuthResponse> {
    const res = await fetch(`${SERVER_BASE}/api/auth/guest`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ displayName }),
    });
    if (!res.ok) {
      throw new Error(`Auth failed: ${await res.text()}`);
    }
    return res.json();
  },

  async listLobbies(): Promise<MatchSummary[]> {
    const res = await fetch(`${SERVER_BASE}/api/lobby`);
    if (!res.ok) {
      throw new Error(`List lobbies failed: ${await res.text()}`);
    }
    return res.json();
  },

  async createMatch(req: CreateMatchRequest): Promise<{ matchId: string; status: string }> {
    const res = await fetch(`${SERVER_BASE}/api/lobby`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      throw new Error(`Create match failed: ${await res.text()}`);
    }
    return res.json();
  },

  async getMatch(id: string): Promise<MatchSummary> {
    const res = await fetch(`${SERVER_BASE}/api/match/${encodeURIComponent(id)}`);
    if (!res.ok) {
      throw new Error(`Get match failed: ${await res.text()}`);
    }
    return res.json();
  },

  getWebSocketURL(token: string): string {
    const wsProto = SERVER_BASE.startsWith('https') ? 'wss:' : 'ws:';
    const host = SERVER_BASE.replace(/^https?:\/\//, '');
    return `${wsProto}//${host}/ws?token=${encodeURIComponent(token)}`;
  },
};
