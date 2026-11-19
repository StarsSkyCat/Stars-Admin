import type { NamedOption, Status } from '../shared';

import { request } from '#/api/request';
import { crudResource } from '#/api/resource';

export interface PostInput {
  code: string;
  name: string;
  remark: null | string;
  sort: number;
  status: Status;
}

export interface Post extends PostInput {
  id: string;
  createdAt: string;
}

export const posts = crudResource<Post, PostInput>('/sys/posts');
export const getPostOptions = () =>
  request<NamedOption[]>('/sys/posts/options');
