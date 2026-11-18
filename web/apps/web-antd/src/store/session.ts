import type { Tokens } from '#/api/core/auth';

export interface Session extends Tokens {
  id: string;
}

type Lock = <T>(name: string, operation: () => Promise<T> | T) => Promise<T>;
type Listener = (current: null | Session) => void;

export function createSessionManager(
  storage: Pick<Storage, 'getItem' | 'removeItem' | 'setItem'>,
  key: string,
  lock?: Lock,
) {
  const listeners = new Set<Listener>();
  const read = (): null | Session => {
    try {
      const session: null | Session = JSON.parse(
        storage.getItem(key) ?? 'null',
      );
      return session?.id && session.accessToken && session.refreshToken
        ? session
        : null;
    } catch {
      return null;
    }
  };
  const notify = () => {
    const current = read();
    for (const listener of listeners) listener(current);
  };
  const write = (session: null | Session) => {
    if (session) storage.setItem(key, JSON.stringify(session));
    else storage.removeItem(key);
    notify();
  };
  // 写锁只保护存储交换，退出可在刷新请求尚未返回时清除会话。
  const update = <T>(operation: () => T) =>
    lock ? lock(`${key}:write`, operation) : Promise.resolve(operation());

  function clear(id: string, accessToken?: string) {
    return update(() => {
      const current = read();
      if (
        current?.id !== id ||
        (accessToken !== undefined && accessToken !== current.accessToken)
      ) {
        return null;
      }
      write(null);
      return current;
    });
  }

  async function rotate(
    failed: Session,
    refresh: (token: string) => Promise<Tokens>,
  ) {
    if (!lock) {
      await clear(failed.id);
      return null;
    }
    return lock(`${key}:refresh`, async () => {
      const current = read();
      if (current?.id !== failed.id) return null;
      if (current.accessToken !== failed.accessToken) return current;
      let tokens: Tokens;
      try {
        tokens = await refresh(current.refreshToken);
      } catch {
        await clear(current.id, current.accessToken);
        return null;
      }
      return update(() => {
        const latest = read();
        if (latest?.id !== current.id) return null;
        const next = {
          ...tokens,
          id: current.id,
          mustChangePassword:
            latest.mustChangePassword || tokens.mustChangePassword,
        };
        write(next);
        return next;
      });
    });
  }

  return {
    clear,
    notify,
    read,
    requirePasswordChange: (id: string) =>
      update(() => {
        const current = read();
        if (current?.id === id && !current.mustChangePassword) {
          write({ ...current, mustChangePassword: true });
        }
      }),
    rotate,
    start: (session: Session) => update(() => write(session)),
    subscribe(listener: Listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
  };
}

const storageKey = 'stars-admin.antd.session.v1';
const webLock: Lock | undefined = navigator.locks
  ? (name, operation) => navigator.locks.request(name, operation)
  : undefined;

export const sessions = createSessionManager(localStorage, storageKey, webLock);

window.addEventListener('storage', ({ key }) => {
  if (key === storageKey || key === null) sessions.notify();
});
