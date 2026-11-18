import type { NamedOption } from '../shared';

import { request } from '#/api/request';

export const getPostOptions = () =>
  request<NamedOption[]>('/sys/posts/options');
