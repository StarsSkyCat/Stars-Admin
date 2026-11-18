import { publicRequest, request } from '#/api/request';

export interface Tokens {
  accessToken: string;
  expiresIn: number;
  mustChangePassword: boolean;
  refreshToken: string;
  tokenType: 'Bearer';
}

export interface LoginInput {
  captchaCode?: string;
  captchaUuid?: string;
  password: string;
  username: string;
}

export interface Captcha {
  enabled: boolean;
  expiresIn: number;
  image: string;
  uuid: string;
}

export interface Profile {
  id: string;
  menus: ProfileMenu[];
  mustChangePassword: boolean;
  nickname: string;
  permissions: string[];
  roleIds: string[];
  username: string;
}

export interface ProfileMenu {
  children: ProfileMenu[];
  component: null | string;
  icon: null | string;
  id: string;
  name: string;
  path: null | string;
  sort: number;
  type: 1 | 2;
  visible: boolean;
}

export interface PasswordInput {
  newPassword: string;
  oldPassword: string;
}

export function validPassword(value: string) {
  const characters = value.match(/./gsu)?.length ?? 0;
  return characters >= 12 && new TextEncoder().encode(value).length <= 72;
}

export const getCaptcha = (signal?: AbortSignal) =>
  publicRequest<Captcha>('/sys/auth/captcha', { signal });

export const loginApi = (data: LoginInput) =>
  publicRequest<Tokens>('/sys/auth/login', { data, method: 'POST' });

export const profileApi = () =>
  request<Profile>('/sys/auth/profile', { personal: true });

export const changePassword = (data: PasswordInput) =>
  request('/sys/auth/password', { data, method: 'PUT', personal: true });

export const logoutApi = (accessToken: string) =>
  publicRequest('/sys/auth/logout', { method: 'POST' }, accessToken);
