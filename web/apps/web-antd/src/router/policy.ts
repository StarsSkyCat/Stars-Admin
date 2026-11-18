import { LOGIN_PATH } from '@vben/constants';

export const CHANGE_PASSWORD_PATH = '/auth/change-password';
export const HOME_PATH = '/dashboard';
export const PROFILE_ERROR_PATH = '/auth/unavailable';

export function authRedirect(
  path: string,
  authenticated: boolean,
  mustChangePassword: boolean,
) {
  if (!authenticated) return path === LOGIN_PATH ? undefined : LOGIN_PATH;
  if (mustChangePassword)
    return path === CHANGE_PASSWORD_PATH ? undefined : CHANGE_PASSWORD_PATH;
  if (path === LOGIN_PATH || path === CHANGE_PASSWORD_PATH) return HOME_PATH;
  return undefined;
}
