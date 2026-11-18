<script setup lang="ts">
import type { TreeProps } from 'ant-design-vue';

import type { Department } from '#/api/system/departments';
import type { Menu } from '#/api/system/menus';
import type { DataScope, Role } from '#/api/system/roles';

import { computed, ref, shallowRef } from 'vue';

import { useVbenModal } from '@vben/common-ui';

import { Alert, Empty, message, Select, Tree } from 'ant-design-vue';

import { getDepartmentOptions } from '#/api/system/departments';
import { getMenuOptions } from '#/api/system/menus';
import { getRole, updateRoleMenus, updateRoleScope } from '#/api/system/roles';

import { dataScopeOptions, showError } from '../shared';

const emit = defineEmits<{ saved: [] }>();
type TreeDataItem = NonNullable<TreeProps['treeData']>[number];
const mode = ref<'menus' | 'scope'>('menus');
const active = shallowRef<Role>();
const nodes = shallowRef<TreeDataItem[]>([]);
const scope = ref<DataScope>(5);
const checked = ref<string[]>([]);
const saving = ref(false);
const loading = ref(false);
const stale = ref(0);
const title = computed(
  () =>
    `${mode.value === 'menus' ? '菜单授权' : '数据范围'}${active.value ? ` · ${active.value.name}` : ''}`,
);

function menuNodes(
  tree: Menu[],
  allowed: Set<string>,
  disabledParent = false,
): TreeDataItem[] {
  return tree.map((node) => {
    const disabled = disabledParent || node.status !== 1;
    if (!disabled) allowed.add(node.id);
    return {
      children: menuNodes(node.children, allowed, disabled),
      disableCheckbox: disabled,
      key: node.id,
      title: node.perm ? `${node.name}（${node.perm}）` : node.name,
    };
  });
}

function departmentNodes(
  tree: Department[],
  allowed: Set<string>,
): TreeDataItem[] {
  return tree.map((node) => {
    if (node.status === 1) allowed.add(node.id);
    return {
      children: departmentNodes(node.children, allowed),
      disableCheckbox: node.status !== 1,
      key: node.id,
      title: node.name,
    };
  });
}

const onCheck: NonNullable<TreeProps['onCheck']> = (keys) => {
  checked.value = (Array.isArray(keys) ? keys : keys.checked).filter(
    (key): key is string => typeof key === 'string',
  );
};
const [Modal, modal] = useVbenModal({
  async onConfirm() {
    if (!active.value || saving.value) return;
    const ids =
      mode.value === 'scope' && scope.value !== 2 ? [] : checked.value;
    if (ids.length > 100) {
      message.warning('一次最多提交 100 个授权节点');
      return;
    }
    if (mode.value === 'scope' && scope.value === 2 && ids.length === 0) {
      message.warning('请选择至少一个部门');
      return;
    }
    saving.value = true;
    modal.lock();
    try {
      await (mode.value === 'menus'
        ? updateRoleMenus(active.value.id, ids)
        : updateRoleScope(active.value.id, scope.value, ids));
      message.success('角色授权已保存');
      await modal.close();
      emit('saved');
    } catch (error) {
      showError(error);
    } finally {
      saving.value = false;
      modal.unlock();
    }
  },
});

async function open(id: string, next: 'menus' | 'scope') {
  if (loading.value || saving.value || id === '1') return;
  loading.value = true;
  try {
    const allowed = new Set<string>();
    const [role, tree] = await Promise.all([
      getRole(id),
      next === 'menus'
        ? getMenuOptions().then((rows) => menuNodes(rows, allowed))
        : getDepartmentOptions().then((rows) => departmentNodes(rows, allowed)),
    ]);
    const grants = next === 'menus' ? role.menuIds : role.deptIds;
    active.value = role;
    nodes.value = tree;
    scope.value = role.dataScope;
    checked.value = grants.filter((key) => allowed.has(key));
    stale.value = grants.length - checked.value.length;
    mode.value = next;
    modal.open();
  } catch (error) {
    showError(error);
  } finally {
    loading.value = false;
  }
}

defineExpose({ open });
</script>

<template>
  <Modal :title="title" class="w-[min(620px,calc(100vw-24px))]">
    <Alert
      v-if="mode === 'menus'"
      class="mb-3"
      message="每个节点单独授权。用户和部门页面需同时拥有对应的列表权限。"
      show-icon
      type="info"
    />
    <div v-else class="mb-3">
      <label for="role-data-scope" class="mb-2 block">数据范围</label>
      <Select
        id="role-data-scope"
        v-model:value="scope"
        :options="[...dataScopeOptions]"
        class="w-full"
      />
    </div>
    <template v-if="mode === 'menus' || scope === 2">
      <Alert
        v-if="stale"
        :message="`${stale} 项现有授权已停用或失效，保存时将移除。`"
        class="mb-3"
        show-icon
        type="warning"
      />
      <div class="max-h-[50vh] overflow-auto">
        <Tree
          v-if="nodes.length"
          :checked-keys="checked"
          :selectable="false"
          :tree-data="nodes"
          check-strictly
          checkable
          default-expand-all
          @check="onCheck"
        />
        <Empty v-else description="没有可选节点" />
      </div>
    </template>
  </Modal>
</template>
