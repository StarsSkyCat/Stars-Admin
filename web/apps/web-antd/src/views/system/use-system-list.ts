import type { PageQuery } from '#/api/system/shared';

import { onMounted, onUnmounted, ref, shallowRef } from 'vue';

import { errorText } from '#/api/errors';

export function useSystemList<Row, Filters extends object>(
  query: (
    filters: Filters,
    pagination: Required<PageQuery>,
    signal: AbortSignal,
  ) => Promise<{ list: Row[]; total: number }>,
  initialFilters: Filters,
) {
  const rows = shallowRef<Row[]>([]);
  const filters = shallowRef(initialFilters);
  const selected = ref<string[]>([]);
  const loading = ref(false);
  const listError = ref('');
  const page = ref(1);
  const pageSize = ref(20);
  const total = ref(0);
  let pending: AbortController | undefined;

  async function load() {
    pending?.abort();
    const controller = new AbortController();
    pending = controller;
    selected.value = [];
    loading.value = true;
    listError.value = '';
    try {
      const result = await query(
        filters.value,
        { page: page.value, pageSize: pageSize.value },
        controller.signal,
      );
      if (controller.signal.aborted) return;
      rows.value = result.list;
      total.value = result.total;
    } catch (error) {
      if (controller.signal.aborted) return;
      rows.value = [];
      total.value = 0;
      listError.value = errorText(error);
    } finally {
      if (!controller.signal.aborted) loading.value = false;
    }
  }

  function search(next: Filters) {
    filters.value = next;
    page.value = 1;
    return load();
  }

  function changePage(next: number, size: number) {
    page.value = size === pageSize.value ? next : 1;
    pageSize.value = size;
    return load();
  }

  onMounted(load);
  onUnmounted(() => pending?.abort());
  return {
    changePage,
    error: listError,
    filters,
    load,
    loading,
    page,
    pageSize,
    rows,
    search,
    selected,
    total,
  };
}
