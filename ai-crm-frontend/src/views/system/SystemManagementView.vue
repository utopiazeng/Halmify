<template>
  <div class="system-page">
    <template v-if="section !== 'user-center'">
      <section v-if="searchableSections.includes(section)" class="system-search-card">
        <el-form :model="filters" label-width="74px" class="system-search-form">
          <template v-if="section === 'user'">
            <el-form-item :label="t('domain.auto.a1aaf352cb')"><el-input v-model="filters.username" :placeholder="t('domain.auto.ad5f05f1cf')" clearable /></el-form-item>
            <el-form-item :label="t('domain.auto.be4c2616b1')"><el-input v-model="filters.name" :placeholder="t('domain.auto.675b3ee2af')" clearable /></el-form-item>
            <el-form-item :label="t('domain.auto.5a9cc5e891')"><el-input v-model="filters.phone" :placeholder="t('domain.auto.5ecce33363')" clearable /></el-form-item>
            <template v-if="searchExpanded">
              <el-form-item :label="t('domain.auto.fe8aa4ef93')">
                <el-select v-model="filters.gender" :placeholder="t('domain.auto.9362948f8d')" clearable>
                  <el-option :label="t('domain.auto.51625d909c')" value="male" />
                  <el-option :label="t('domain.auto.26bc84961c')" value="female" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('domain.auto.6b26695e4d')">
                <el-select v-model="filters.role_id" :placeholder="t('domain.auto.30741f3cf7')" clearable>
                  <el-option v-for="role in roles" :key="role.id" :label="roleDisplayName(role)" :value="role.id" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('domain.auto.5ff376c0bf')">
                <el-tree-select v-model="filters.department_id" :data="departmentSelectTree" check-strictly clearable :placeholder="t('domain.auto.50210083eb')" />
              </el-form-item>
              <el-form-item :label="t('domain.auto.552f0fe210')">
                <el-select v-model="filters.post_id" :placeholder="t('domain.auto.dc5de8e845')" clearable>
                  <el-option v-for="post in posts" :key="post.id" :label="post.name" :value="post.id" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('domain.auto.6102c5c9e6')"><el-date-picker v-model="filters.date_range" type="daterange" :start-placeholder="t('domain.auto.1f29196891')" :end-placeholder="t('domain.auto.f4b9b2b5de')" value-format="YYYY-MM-DD" /></el-form-item>
            </template>
          </template>

          <template v-else-if="section === 'role'">
            <el-form-item :label="t('domain.auto.3aa1f085b2')"><el-input v-model="filters.name" :placeholder="t('domain.auto.b7c17b9e6e')" clearable /></el-form-item>
            <el-form-item :label="t('domain.auto.c12ace673d')"><el-input v-model="filters.code" :placeholder="t('domain.auto.67819cee9b')" clearable /></el-form-item>
            <el-form-item :label="t('domain.auto.a0a62a0031')"><el-input v-model="filters.description" :placeholder="t('domain.auto.0c205e28ad')" clearable /></el-form-item>
            <template v-if="searchExpanded">
              <el-form-item :label="t('domain.auto.c665191612')">
                <el-select v-model="filters.status" :placeholder="t('domain.auto.dba277df58')" clearable>
                  <el-option :label="t('domain.auto.d4e9ca3dd4')" value="active" />
                  <el-option :label="t('domain.auto.be70be5a2e')" value="disabled" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('domain.auto.6102c5c9e6')"><el-date-picker v-model="filters.date_range" type="daterange" :start-placeholder="t('domain.auto.1f29196891')" :end-placeholder="t('domain.auto.f4b9b2b5de')" value-format="YYYY-MM-DD" /></el-form-item>
            </template>
          </template>

          <template v-else-if="isLogSection">
            <el-form-item :label="t('domain.auto.ae0849ff19')"><el-input v-model="filters.user" :placeholder="t('domain.auto.012751d048')" clearable /></el-form-item>
            <el-form-item :label="t('domain.auto.62e951a692')">
              <el-select v-model="filters.result" :placeholder="t('domain.auto.dba277df58')" clearable>
                <el-option :label="t('domain.auto.51991a5d11')" value="success" />
                <el-option :label="t('domain.auto.3e3c8068bb')" value="failed" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.b6fed9af83')"><el-date-picker v-model="filters.date_range" type="daterange" :start-placeholder="t('domain.auto.1f29196891')" :end-placeholder="t('domain.auto.f4b9b2b5de')" value-format="YYYY-MM-DD" /></el-form-item>
          </template>

          <div class="system-search-actions">
            <el-button @click="resetFilters">{{ t('domain.auto.3d81345303') }}</el-button>
            <el-button type="primary" @click="loadPage">{{ t('domain.auto.711363c424') }}</el-button>
            <el-button v-if="section === 'user' || section === 'role'" link type="primary" @click="searchExpanded = !searchExpanded">
              {{ searchExpanded ? t('domain.auto.5d5815647c') : t('domain.auto.b0e24833f7') }}
            </el-button>
          </div>
        </el-form>
      </section>

      <section ref="tableCardRef" class="system-card" :class="{ 'system-card-fullscreen': isFullscreen }">
        <div class="system-toolbar">
          <div class="toolbar-left">
            <el-button v-if="['user','role','department','post','menu'].includes(section)" type="primary" plain @click="openCreate">
              {{ createButtonText }}
            </el-button>
            <el-button v-if="section === 'department' || section === 'menu'" @click="expandAll = !expandAll">{{ expandAll ? t('domain.auto.5d5815647c') : t('domain.auto.b0e24833f7') }}</el-button>
            <el-button v-if="isLogSection" @click="exportLogs">{{ t('domain.auto.188896795f') }}</el-button>
          </div>
          <div class="toolbar-icons">
            <el-tooltip :content="t('domain.auto.f04090805c')"><button class="system-icon-btn" :aria-label="t('domain.auto.f04090805c')" @click="searchExpanded = !searchExpanded"><Icon icon="ri:search-line" /></button></el-tooltip>
            <el-tooltip :content="t('domain.auto.38108eaa1d')"><button class="system-icon-btn" :aria-label="t('domain.auto.38108eaa1d')" @click="loadPage"><Icon icon="ri:refresh-line" /></button></el-tooltip>
            <el-tooltip v-if="sortableSections.includes(section)" :content="sortTooltip"><button class="system-icon-btn" :aria-label="t('domain.auto.dc35af8d69')" @click="toggleSort"><Icon :icon="sortIcon" /></button></el-tooltip>
            <el-tooltip :content="isFullscreen ? t('domain.auto.a170c537d0') : t('domain.auto.93c44f6b1b')"><button class="system-icon-btn" :aria-label="isFullscreen ? t('domain.auto.a170c537d0') : t('domain.auto.93c44f6b1b')" @click="toggleFullscreen"><Icon :icon="isFullscreen ? 'ri:fullscreen-exit-line' : 'ri:fullscreen-line'" /></button></el-tooltip>
            <el-tooltip v-if="columnSections.includes(section)" :content="t('domain.auto.56292a48ef')"><button class="system-icon-btn" :aria-label="t('domain.auto.56292a48ef')" @click="openColumnSettings"><Icon icon="ri:settings-3-line" /></button></el-tooltip>
          </div>
        </div>

        <el-table
          v-if="section === 'user'"
          :data="userRows"
          class="system-table"
          :key="columnSignature"
          row-key="id"
          v-loading="loading"
        >
          <el-table-column type="selection" width="42" />
          <el-table-column type="index" :label="t('domain.auto.6cd7c92cbd')" width="70" />
          <el-table-column v-if="visibleColumn('username')" prop="username" :label="t('domain.auto.a1aaf352cb')" min-width="120" />
          <el-table-column v-if="visibleColumn('name')" prop="name" :label="t('domain.auto.be4c2616b1')" min-width="120"><template #default="{ row }">{{ row.name || '-' }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('gender')" prop="gender" :label="t('domain.auto.fe8aa4ef93')" width="90"><template #default="{ row }">{{ row.gender || '-' }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('phone')" prop="phone" :label="t('domain.auto.5a9cc5e891')" min-width="130"><template #default="{ row }">{{ row.phone || '-' }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('roles')" :label="t('domain.auto.e323915cc2')" min-width="160"><template #default="{ row }"><span>{{ roleNames(row.role_ids) || '-' }}</span><el-tag v-if="row.is_primary_admin" class="primary-admin-tag" size="small" type="warning">{{ t('system.user.primaryAdmin') }}</el-tag></template></el-table-column>
          <el-table-column v-if="visibleColumn('department')" :label="t('domain.auto.5ff376c0bf')" min-width="130"><template #default="{ row }">{{ deptName(row.department_id) || '-' }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('post')" :label="t('domain.auto.552f0fe210')" min-width="130"><template #default="{ row }">{{ postName(row.post_id) || '-' }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('created_at')" :label="t('domain.auto.6102c5c9e6')" min-width="150"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('updated_at')" :label="t('domain.auto.beef2c086d')" min-width="150"><template #default="{ row }">{{ formatTime(row.updated_at) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.f3ea6d345e')" width="132" fixed="right">
            <template #default="{ row }">
              <el-dropdown trigger="click">
                <button class="more-btn"><Icon icon="ri:more-2-fill" /></button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="openEdit(row)">{{ t('domain.auto.a7f814c0a4') }}</el-dropdown-item>
                    <el-dropdown-item :disabled="row.is_primary_admin" @click="row.is_primary_admin ? null : openResetPassword(row)">{{ t('system.user.resetPassword') }}</el-dropdown-item>
                    <el-dropdown-item :disabled="row.is_primary_admin" @click="row.is_primary_admin ? null : toggleUserStatus(row)">{{ row.status === 'disabled' ? t('system.user.enableAccount') : t('system.user.disableAccount') }}</el-dropdown-item>
                    <el-dropdown-item divided :disabled="row.is_primary_admin" @click="row.is_primary_admin ? null : confirmDelete(t('domain.auto.9ba763ea34'), row.username, () => deleteUser(row.id))">{{ t('domain.auto.3755f56f2f') }}</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </template>
          </el-table-column>
        </el-table>

        <el-table v-else-if="section === 'role'" :data="roleRows" class="system-table" :key="columnSignature" row-key="id" v-loading="loading">
          <el-table-column v-if="visibleColumn('id')" prop="id" :label="t('domain.auto.c38a312709')" width="90" />
          <el-table-column v-if="visibleColumn('name')" :label="t('domain.auto.3aa1f085b2')" min-width="150"><template #default="{ row }">{{ roleDisplayName(row) }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('code')" prop="code" :label="t('domain.auto.c12ace673d')" min-width="160" />
          <el-table-column v-if="visibleColumn('description')" :label="t('domain.auto.a0a62a0031')" min-width="220" show-overflow-tooltip><template #default="{ row }">{{ roleDisplayDescription(row) }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('status')" :label="t('domain.auto.c665191612')" width="120"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
          <el-table-column v-if="visibleColumn('created_at')" :label="t('domain.auto.6102c5c9e6')" min-width="160"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('updated_at')" :label="t('domain.auto.9b9857e025')" min-width="160"><template #default="{ row }">{{ formatTime(row.updated_at) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.f3ea6d345e')" width="90" fixed="right">
            <template #default="{ row }">
              <el-dropdown trigger="click">
                <button class="more-btn"><Icon icon="ri:more-2-fill" /></button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="openEdit(row)">{{ t('domain.auto.a7f814c0a4') }}</el-dropdown-item>
                    <el-dropdown-item @click="openPermissionDrawer(row)">{{ t('domain.auto.1a933237a8') }}</el-dropdown-item>
                    <el-dropdown-item @click="toggleRole(row)">{{ row.status === 'disabled' ? t('domain.auto.d4e9ca3dd4') : t('domain.auto.be70be5a2e') }}</el-dropdown-item>
                    <el-dropdown-item divided @click="confirmDelete(t('domain.auto.6b26695e4d'), row.name, () => deleteRole(row.id))">{{ t('domain.auto.3755f56f2f') }}</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </template>
          </el-table-column>
        </el-table>

        <el-table v-else-if="section === 'department'" :data="departmentRows" class="system-table" :key="columnSignature" row-key="id" :default-expand-all="expandAll" v-loading="loading">
          <el-table-column v-if="visibleColumn('name')" prop="name" :label="t('domain.auto.a35c1d2eb8')" min-width="180" />
          <el-table-column v-if="visibleColumn('code')" prop="code" :label="t('domain.auto.e43f2be96c')" min-width="120" />
          <el-table-column v-if="visibleColumn('leader')" prop="leader" :label="t('domain.auto.974d383f36')" min-width="120"><template #default="{ row }">{{ row.leader || '-' }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('phone')" prop="phone" :label="t('domain.auto.e02f6e5760')" min-width="130"><template #default="{ row }">{{ row.phone || '-' }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('email')" prop="email" :label="t('domain.auto.9ed627bcf6')" min-width="160"><template #default="{ row }">{{ row.email || '-' }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('sort')" prop="sort" :label="t('domain.auto.dc35af8d69')" width="90" />
          <el-table-column v-if="visibleColumn('status')" :label="t('domain.auto.62e951a692')" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
          <el-table-column v-if="visibleColumn('updated_at')" :label="t('domain.auto.093dea88c9')" min-width="160"><template #default="{ row }">{{ formatTime(row.updated_at) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.f3ea6d345e')" width="168" fixed="right">
            <template #default="{ row }">
              <div class="row-actions three">
                <button class="action-btn add" @click="openCreate(row)"><Icon icon="ri:add-line" /></button>
                <button class="action-btn edit" @click="openEdit(row)"><Icon icon="ri:pencil-line" /></button>
                <button class="action-btn danger" @click="confirmDelete(t('domain.auto.91061a56c0'), row.name, () => deleteDepartment(row.id))"><Icon icon="ri:delete-bin-line" /></button>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <el-table v-else-if="section === 'post'" :data="postRows" class="system-table" :key="columnSignature" row-key="id" v-loading="loading">
          <el-table-column type="index" :label="t('domain.auto.6cd7c92cbd')" width="80" />
          <el-table-column v-if="visibleColumn('name')" prop="name" :label="t('domain.auto.f7e2b452ba')" min-width="180" />
          <el-table-column v-if="visibleColumn('code')" prop="code" :label="t('domain.auto.0fc3644a4f')" min-width="160" />
          <el-table-column v-if="visibleColumn('sort')" prop="sort" :label="t('domain.auto.dc35af8d69')" width="90" />
          <el-table-column v-if="visibleColumn('status')" :label="t('domain.auto.62e951a692')" width="110"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
          <el-table-column v-if="visibleColumn('remark')" prop="remark" :label="t('domain.auto.e0361480e3')" min-width="180"><template #default="{ row }">{{ row.remark || '-' }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('updated_at')" :label="t('domain.auto.093dea88c9')" min-width="160"><template #default="{ row }">{{ formatTime(row.updated_at) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.f3ea6d345e')" width="118" fixed="right">
            <template #default="{ row }">
              <div class="row-actions">
                <button class="action-btn edit" @click="openEdit(row)"><Icon icon="ri:pencil-line" /></button>
                <button class="action-btn danger" @click="confirmDelete(t('domain.auto.f0f3e908c2'), row.name, () => deletePost(row.id))"><Icon icon="ri:delete-bin-line" /></button>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <el-table v-else-if="section === 'menu'" :data="menuRows" class="system-table" :key="columnSignature" row-key="id" :default-expand-all="expandAll" v-loading="loading">
          <el-table-column v-if="visibleColumn('name')" prop="name" :label="t('domain.auto.2637dede68')" min-width="180" />
          <el-table-column v-if="visibleColumn('type')" :label="t('domain.auto.7907c3c545')" width="120"><template #default="{ row }"><el-tag :type="menuTagType(row.type)" effect="light">{{ menuTypeLabel(row.type) }}</el-tag></template></el-table-column>
          <el-table-column v-if="visibleColumn('path')" prop="path" :label="t('domain.auto.19da0c94ef')" min-width="160" show-overflow-tooltip />
          <el-table-column v-if="visibleColumn('permission')" prop="permission" :label="t('domain.auto.18a0035e57')" min-width="170" show-overflow-tooltip />
          <el-table-column v-if="visibleColumn('sort')" prop="sort" :label="t('domain.auto.dc35af8d69')" width="90" />
          <el-table-column v-if="visibleColumn('updated_at')" :label="t('domain.auto.0f92576ac5')" min-width="150"><template #default="{ row }">{{ formatTime(row.updated_at) }}</template></el-table-column>
          <el-table-column v-if="visibleColumn('status')" :label="t('domain.auto.62e951a692')" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
          <el-table-column :label="t('domain.auto.f3ea6d345e')" width="168" fixed="right">
            <template #default="{ row }">
              <div class="row-actions three">
                <button class="action-btn add" @click="openCreate(row)"><Icon icon="ri:add-line" /></button>
                <button class="action-btn edit" @click="openEdit(row)"><Icon icon="ri:pencil-line" /></button>
                <button class="action-btn danger" @click="confirmDelete(t('domain.auto.7082456207'), row.name, () => deleteMenu(row.id))"><Icon icon="ri:delete-bin-line" /></button>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <el-table v-else :data="logRows" class="system-table compact-log-table" row-key="id" v-loading="loading">
          <el-table-column :label="t('domain.auto.8cac83c83b')" min-width="110"><template #default="{ row }">{{ logRowPrefix + row.id }}</template></el-table-column>
          <el-table-column v-if="section === 'login-log'" prop="username" :label="t('domain.auto.a311ed747b')" min-width="110" />
          <el-table-column v-if="section === 'login-log'" :label="t('domain.auto.47a991d18c')" width="100"><template #default>{{ t('system.device.pc') }}</template></el-table-column>
          <el-table-column v-if="section === 'login-log'" prop="ip" :label="t('domain.auto.67d2d7970f')" min-width="120" show-overflow-tooltip />
          <el-table-column v-if="section === 'login-log'" :label="t('domain.auto.3f0cc5580f')" min-width="140"><template #default="{ row }">{{ osFromUA(row.user_agent) }} / {{ browserFromUA(row.user_agent) }}</template></el-table-column>
          <template v-if="section !== 'login-log'">
            <el-table-column :label="t('domain.auto.f84ec364a6')" min-width="130" show-overflow-tooltip><template #default="{ row }">{{ logModuleLabel(row) }}</template></el-table-column>
            <el-table-column prop="operation" :label="t('domain.auto.19e41f1bea')" min-width="120"><template #default="{ row }"><el-tag effect="light">{{ operationLabel(row.operation) }}</el-tag></template></el-table-column>
            <el-table-column prop="username" :label="t('domain.auto.ae0849ff19')" min-width="110" />
            <el-table-column prop="ip" :label="t('domain.auto.43bddfdb71')" min-width="120" show-overflow-tooltip />
          </template>
          <el-table-column :label="t('domain.auto.62e951a692')" width="100"><template #default="{ row }"><el-tag type="success" effect="light">{{ row.result === 'failed' ? t('common.status.failed') : t('common.status.success') }}</el-tag></template></el-table-column>
          <el-table-column prop="reason" :label="t('domain.auto.412f54dc38')" min-width="150" show-overflow-tooltip><template #default="{ row }">{{ logDescription(row) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.219b76faca')" min-width="150"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.f3ea6d345e')" width="90" fixed="right"><template #default="{ row }"><button class="action-btn view" @click="showLog(row)"><Icon icon="ri:eye-line" /></button></template></el-table-column>
        </el-table>

        <div v-if="['user','post','log','login-log','security-log'].includes(section)" class="system-pagination">
          <span>{{ t('domain.auto.3b6ef811b8') }} {{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</span>
          <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes, jumper" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" />
        </div>
      </section>
    </template>

    <UserCenter v-else :roles="roles" />

    <el-dialog v-model="dialog.visible" :title="dialog.title" :width="dialogWidth" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="92px" class="system-dialog-form" :class="{ 'role-dialog-form': section === 'role' }">
        <template v-if="section === 'user'">
          <el-form-item :label="t('domain.auto.a1aaf352cb')" prop="username"><el-input v-model="form.username" :disabled="dialog.mode === 'edit'" :placeholder="t('domain.auto.ad5f05f1cf')" /></el-form-item>
          <el-form-item v-if="dialog.mode === 'create'" :label="t('domain.auto.c839a8ff17')" prop="password"><el-input v-model="form.password" show-password :placeholder="t('domain.auto.713b738250')" /></el-form-item>
          <el-form-item :label="t('domain.auto.be4c2616b1')" prop="name"><el-input v-model="form.name" :placeholder="t('domain.auto.e28d9f287b')" /></el-form-item>
          <el-form-item :label="t('domain.auto.fe8aa4ef93')" prop="gender"><el-select v-model="form.gender"><el-option :label="t('domain.auto.51625d909c')" value="male" /><el-option :label="t('domain.auto.26bc84961c')" value="female" /></el-select></el-form-item>
          <el-form-item :label="t('domain.auto.6b26695e4d')" prop="role_ids"><el-select v-model="form.role_ids" multiple :placeholder="t('domain.auto.382f4b5559')"><el-option v-for="role in roles" :key="role.id" :label="roleDisplayName(role)" :value="role.id" /></el-select></el-form-item>
          <el-form-item :label="t('domain.auto.91061a56c0')"><el-tree-select v-model="form.department_id" :data="departmentSelectTree" check-strictly clearable :placeholder="t('domain.auto.50210083eb')" /></el-form-item>
          <el-form-item :label="t('domain.auto.f0f3e908c2')"><el-select v-model="form.post_id" clearable :placeholder="t('domain.auto.dc5de8e845')"><el-option v-for="post in posts" :key="post.id" :label="post.name" :value="post.id" /></el-select></el-form-item>
          <el-form-item :label="t('domain.auto.5a9cc5e891')"><el-input v-model="form.phone" :placeholder="t('domain.auto.5ecce33363')" /></el-form-item>
          <el-form-item v-if="dialog.mode === 'edit'" :label="t('system.user.accountStatus')">
            <el-switch v-model="form.enabled" :disabled="form.is_primary_admin" :active-text="t('system.user.enabled')" :inactive-text="t('system.user.disabled')" />
            <div v-if="form.is_primary_admin" class="form-help">{{ t('system.user.primaryAdminProtected') }}</div>
          </el-form-item>
        </template>

        <template v-else-if="section === 'role'">
          <el-alert
            class="role-admin-note"
            type="info"
            :closable="false"
            show-icon
            :title="t('domain.auto.8254fb6451')"
          />
          <el-form-item :label="t('domain.auto.44c7a7fd74')">
            <el-radio-group v-model="form.role_preset" @change="applyRolePreset">
              <el-radio-button label="normal">{{ t('domain.auto.e357fc952d') }}</el-radio-button>
              <el-radio-button label="salesperson">{{ t('system.rolePreset.salesperson') }}</el-radio-button>
              <el-radio-button label="supervisor">{{ t('domain.auto.f7751161e4') }}</el-radio-button>
              <el-radio-button label="sdr_supervisor">{{ t('domain.auto.396d9b8c30') }}</el-radio-button>
            </el-radio-group>
            <div class="form-help">{{ t('domain.auto.4bb2e0adcb') }}</div>
          </el-form-item>
          <el-form-item :label="t('domain.auto.3aa1f085b2')" prop="name"><el-input v-model="form.name" :placeholder="t('domain.auto.b7c17b9e6e')" /></el-form-item>
          <el-form-item :label="t('domain.auto.c12ace673d')" prop="code">
            <el-input v-model="form.code" :placeholder="t('domain.auto.fd8c0dde7f')" />
            <div class="form-help">{{ t('domain.auto.698957986d') }}</div>
          </el-form-item>
          <el-form-item :label="t('domain.auto.412f54dc38')"><el-input v-model="form.description" type="textarea" :rows="3" :placeholder="t('domain.auto.0c205e28ad')" /></el-form-item>
          <el-form-item :label="t('domain.auto.d4e9ca3dd4')"><el-switch v-model="form.enabled" /></el-form-item>
        </template>

        <template v-else-if="section === 'department'">
          <el-form-item :label="t('domain.auto.7cd8969ba8')"><el-tree-select v-model="form.parent_id" :data="departmentSelectTree" check-strictly clearable :placeholder="t('domain.auto.7e865254af')" /></el-form-item>
          <el-form-item :label="t('domain.auto.a35c1d2eb8')" prop="name"><el-input v-model="form.name" :placeholder="t('domain.auto.95e96ffe8a')" /></el-form-item>
          <el-form-item :label="t('domain.auto.e43f2be96c')" prop="code"><el-input v-model="form.code" :placeholder="t('domain.auto.1bb5067274')" /></el-form-item>
          <el-form-item :label="t('domain.auto.974d383f36')">
            <el-select
              v-model="form.leader"
              filterable
              allow-create
              default-first-option
              clearable
              :reserve-keyword="true"
              :placeholder="t('domain.auto.8b134548e7')"
            >
              <el-option
                v-for="item in leaderOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('domain.auto.e02f6e5760')"><el-input v-model="form.phone" :placeholder="t('domain.auto.4597687d8a')" /></el-form-item>
          <el-form-item :label="t('domain.auto.9ed627bcf6')"><el-input v-model="form.email" :placeholder="t('domain.auto.f2dc24deb8')" /></el-form-item>
          <el-form-item :label="t('domain.auto.dc35af8d69')"><el-input-number v-model="form.sort" :min="1" class="full-input" /></el-form-item>
          <el-form-item :label="t('domain.auto.62e951a692')"><el-switch v-model="form.enabled" /></el-form-item>
          <el-form-item :label="t('domain.auto.e0361480e3')" class="wide-field"><el-input v-model="form.remark" type="textarea" :rows="3" :placeholder="t('domain.auto.57e709d957')" /></el-form-item>
        </template>

        <template v-else-if="section === 'post'">
          <el-form-item :label="t('domain.auto.f7e2b452ba')" prop="name"><el-input v-model="form.name" :placeholder="t('domain.auto.a13301018c')" /></el-form-item>
          <el-form-item :label="t('domain.auto.0fc3644a4f')" prop="code"><el-input v-model="form.code" :placeholder="t('domain.auto.6524110e2e')" /></el-form-item>
          <el-form-item :label="t('domain.auto.dc35af8d69')"><el-input-number v-model="form.sort" :min="1" class="full-input" /></el-form-item>
          <el-form-item :label="t('domain.auto.62e951a692')"><el-switch v-model="form.enabled" /></el-form-item>
          <el-form-item :label="t('domain.auto.e0361480e3')"><el-input v-model="form.remark" type="textarea" :rows="3" :placeholder="t('domain.auto.57e709d957')" /></el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitForm">{{ t('domain.auto.09cbc97ae2') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="resetPasswordDialog.visible" :title="t('system.user.resetPassword')" width="420px" destroy-on-close>
    <el-form label-position="top" class="system-dialog-form">
      <el-form-item :label="t('domain.auto.a1aaf352cb')">
        <el-input :model-value="resetPasswordDialog.username" disabled />
      </el-form-item>
      <el-form-item :label="t('system.user.newPassword')" required>
        <el-input v-model="resetPasswordDialog.password" show-password :placeholder="t('system.user.newPasswordPlaceholder')" />
        <div class="form-help">{{ t('system.user.passwordRule') }}</div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="resetPasswordDialog.visible = false">{{ t('common.action.cancel') }}</el-button>
      <el-button type="primary" :loading="resetPasswordDialog.saving" @click="submitResetPassword">{{ t('common.action.submit') }}</el-button>
    </template>
  </el-dialog>

    <el-drawer v-model="menuDrawer" :title="dialog.title" size="44%" destroy-on-close>
      <el-form ref="menuFormRef" :model="form" :rules="rules" label-width="96px" class="menu-drawer-form">
        <el-form-item :label="t('domain.auto.e4e46c7235')">
          <el-radio-group v-model="form.type">
            <el-radio-button label="catalog">{{ t('domain.auto.41e5243e2d') }}</el-radio-button>
            <el-radio-button label="menu">{{ t('domain.auto.7082456207') }}</el-radio-button>
            <el-radio-button label="button">{{ t('domain.auto.a6a6eaa2f1') }}</el-radio-button>
            <el-radio-button label="iframe">{{ t('domain.auto.1e8496ff67') }}</el-radio-button>
            <el-radio-button label="external">{{ t('domain.auto.9f0021402a') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('domain.auto.fafc8d9d72')"><el-tree-select v-model="form.parent_id" :data="menuSelectTree" check-strictly clearable :placeholder="t('domain.auto.2002977f6f')" /></el-form-item>
        <el-form-item :label="t('domain.auto.b4c0418914')" prop="name"><el-input v-model="form.name" :placeholder="t('domain.auto.b4c0418914')" /></el-form-item>
        <el-form-item :label="t('domain.auto.54faad116a')"><el-input v-model="form.path" :placeholder="t('domain.auto.951198fef5')" /></el-form-item>
        <el-form-item :label="t('domain.auto.18a0035e57')"><el-input v-model="form.permission" :placeholder="t('domain.auto.7c5e9c58b5')" /></el-form-item>
        <el-form-item :label="t('domain.auto.1f24c1e5aa')"><el-input v-model="form.icon" :placeholder="t('domain.auto.fadb3075b0')" /></el-form-item>
        <el-form-item :label="t('domain.auto.c0aace1e23')"><el-input-number v-model="form.sort" :min="1" class="full-input" /></el-form-item>
        <el-divider>{{ t('domain.auto.482e1ba326') }}</el-divider>
        <el-form-item :label="t('domain.auto.3079bc7a69')"><el-switch v-model="form.enabled" /></el-form-item>
        <el-form-item :label="t('domain.auto.23a8bb1dd7')"><el-switch v-model="form.hidden" /></el-form-item>
        <el-form-item :label="t('domain.auto.aee99fe408')"><el-switch v-model="form.show_badge" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="menuDrawer = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitForm">{{ t('domain.auto.f526c89937') }}</el-button>
      </template>
    </el-drawer>

    <el-drawer v-model="columnDrawer" :title="t('domain.auto.56292a48ef')" size="320px">
      <div class="column-settings">
        <div class="column-settings-head">
          <el-button link type="primary" @click="selectAllColumns">{{ t('domain.auto.3e44b2a933') }}</el-button>
          <el-button link type="primary" @click="resetColumns">{{ t('domain.auto.a191935bc6') }}</el-button>
        </div>
        <el-checkbox-group v-model="columnSelections[section]" class="column-checkboxes" @change="saveColumnSettings">
          <el-checkbox v-for="item in currentColumnOptions" :key="item.key" :label="item.key">{{ item.label }}</el-checkbox>
        </el-checkbox-group>
      </div>
    </el-drawer>

    <el-drawer v-model="permissionDrawer.visible" :title="permissionDrawerTitle" size="min(760px, 94vw)" destroy-on-close>
      <div class="permission-drawer">
        <el-alert
          type="info"
          :closable="false"
          show-icon
          :title="t('domain.auto.70c2e47091')"
        />
        <div class="permission-toolbar">
          <el-button size="small" @click="selectAllPermissions">{{ t('domain.auto.b93ad0b605') }}</el-button>
          <el-button size="small" @click="clearAllPermissions">{{ t('domain.auto.a6d3de0e86') }}</el-button>
        </div>
        <section class="permission-section">
          <div class="permission-section-title">
            <strong>{{ t('domain.auto.283de88009') }}</strong>
            <span>{{ t('domain.auto.2ae0a13761') }}</span>
          </div>
          <el-checkbox-group v-model="permissionDrawer.selected" class="permission-grid">
            <div v-for="item in businessPermissionOptions" :key="item.resource" class="permission-resource-card">
              <div class="permission-resource-head">
                <strong>{{ item.label }}</strong>
                <span>{{ item.hint }}</span>
              </div>
              <div class="permission-action-grid">
                <el-checkbox v-for="action in item.actions" :key="`${item.resource}:${action}`" :label="`${item.resource}:${action}`" class="permission-action-option">
                  <span class="permission-action-compact">
                    <strong>{{ actionLabel(action) }}</strong>
                    <el-tooltip :content="permissionTooltipText(action, item.label)" placement="top" effect="light">
                      <Icon icon="ri:question-line" />
                    </el-tooltip>
                  </span>
                </el-checkbox>
              </div>
            </div>
          </el-checkbox-group>
        </section>
        <section class="permission-section">
          <div class="permission-section-title">
            <strong>{{ t('domain.auto.144c7a3dbc') }}</strong>
            <span>{{ t('domain.auto.c9f3ea1adc') }}</span>
          </div>
          <el-checkbox-group v-model="permissionDrawer.selected" class="permission-grid">
            <div v-for="item in platformPermissionOptions" :key="item.resource" class="permission-resource-card">
              <div class="permission-resource-head">
                <strong>{{ item.label }}</strong>
                <span>{{ item.hint }}</span>
              </div>
              <div class="permission-action-grid">
                <el-checkbox v-for="action in item.actions" :key="`${item.resource}:${action}`" :label="`${item.resource}:${action}`" class="permission-action-option">
                  <span class="permission-action-compact">
                    <strong>{{ actionLabel(action) }}</strong>
                    <el-tooltip :content="permissionTooltipText(action, item.label)" placement="top" effect="light">
                      <Icon icon="ri:question-line" />
                    </el-tooltip>
                  </span>
                </el-checkbox>
              </div>
            </div>
          </el-checkbox-group>
        </section>
      </div>
      <template #footer>
        <el-button :disabled="permissionDrawer.saving" @click="permissionDrawer.visible = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" :loading="permissionDrawer.saving" @click="saveRolePermissions">{{ t('domain.auto.d5355d17be') }}</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, defineComponent, h, onBeforeUnmount, onMounted, reactive, ref, resolveComponent, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../api'
import { enabledModules as readEnabledModules, hasPermission } from '../../permissions'
import { resolveProtectedImageUrl, revokeObjectUrl } from '../../utils/protectedImage'
import { apiErrorMessage } from '../../utils/api-error'

const StatusTag = defineComponent({
  props: { status: { type: String, default: 'active' } },
  setup(props) {
    return () => h('span', { class: ['status-pill', props.status === 'disabled' ? 'off' : 'on'] }, props.status === 'disabled' ? t('common.status.disabled') : t('common.status.enabled'))
  }
})

const UserCenter = defineComponent({
  props: { roles: { type: Array, default: () => [] } },
  setup(props) {
    const profile = reactive({ username: '', name: '', gender: 'male', email: '', phone: '', address: '', bio: '', avatar_url: '', cover_url: '', department_id: undefined, post_id: undefined, department_name: '', post_name: '', role_ids: [] })
    const password = reactive({ current_password: '', new_password: '', confirm_password: '' })
    const sessions = ref([])
    const currentSessionId = ref('')
    const departments = ref([])
    const posts = ref([])
    const avatarUploading = ref(false)
    const avatarDisplayUrl = ref('')
    const sessionPage = reactive({ page: 1, size: 5, total: 0 })
    const systemAvatarOptions = [
      { key: 'blue', label: t('profile.avatar.blue'), value: systemAvatarDataUrl('#2563eb', '#60a5fa', 'U') },
      { key: 'green', label: t('profile.avatar.green'), value: systemAvatarDataUrl('#059669', '#34d399', 'C') },
      { key: 'orange', label: t('profile.avatar.orange'), value: systemAvatarDataUrl('#ea580c', '#fbbf24', 'S') },
      { key: 'purple', label: t('profile.avatar.purple'), value: systemAvatarDataUrl('#7c3aed', '#c084fc', 'A') },
      { key: 'slate', label: t('profile.avatar.slate'), value: systemAvatarDataUrl('#334155', '#94a3b8', 'M') },
      { key: 'rose', label: t('profile.avatar.rose'), value: systemAvatarDataUrl('#e11d48', '#fb7185', 'P') }
    ]

    const roleNames = computed(() => (profile.role_ids || []).map((id) => props.roles.find((r) => r.id === id)?.name).filter(Boolean))

    async function load() {
      const [profileRes, sessionRes, deptRes, postRes] = await Promise.allSettled([
        api.get('/api/v1/accounts/me/profile'),
        api.get('/api/v1/accounts/me/sessions', { params: { page: sessionPage.page, size: sessionPage.size } }),
        hasPermission('system:department:view') ? api.get('/api/v1/system/departments') : Promise.resolve({ data: { items: [] } }),
        hasPermission('system:post:view') ? api.get('/api/v1/system/posts', { params: { page: 1, size: 200 } }) : Promise.resolve({ data: { items: [] } })
      ])
      if (profileRes.status === 'rejected') throw profileRes.reason
      if (sessionRes.status === 'rejected') throw sessionRes.reason
      Object.assign(profile, profileRes.value.data || {})
      sessions.value = sessionRes.value.data?.items || []
      sessionPage.total = sessionRes.value.data?.total || sessions.value.length
      currentSessionId.value = sessionRes.value.data?.current_session_id || ''
      departments.value = deptRes.status === 'fulfilled' ? (deptRes.value.data?.items || []) : []
      posts.value = postRes.status === 'fulfilled' ? (postRes.value.data?.items || []) : []
    }
    async function refreshAvatarDisplay() {
      const source = String(profile.avatar_url || '').trim()
      const previous = avatarDisplayUrl.value
      if (!source) {
        avatarDisplayUrl.value = ''
        revokeObjectUrl(previous)
        return
      }
      try {
        const next = await resolveProtectedImageUrl(source)
        avatarDisplayUrl.value = next
        if (previous && previous !== next) revokeObjectUrl(previous)
      } catch {
        avatarDisplayUrl.value = ''
        revokeObjectUrl(previous)
      }
    }
    async function saveProfile() {
      await api.put('/api/v1/accounts/me/profile', profilePayload())
      ElMessage.success(t('common.message.opSuccess'))
      await load()
      syncCurrentUser()
    }
    async function uploadAvatar(file) {
      const raw = file?.raw || file
      if (!raw) return false
      if (!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(raw.type)) {
        ElMessage.error(t('im.avatarTypeInvalid'))
        return false
      }
      if (raw.size > 5 * 1024 * 1024) {
        ElMessage.error(t('im.avatarTooLarge'))
        return false
      }
      const formData = new FormData()
      formData.append('file', raw)
      avatarUploading.value = true
      try {
        const res = await api.post('/api/v1/accounts/me/avatar', formData, {
          headers: {}
        })
        Object.assign(profile, res.data || {})
        await refreshAvatarDisplay()
        syncCurrentUser()
        ElMessage.success(t('common.message.opSuccess'))
      } catch (error) {
        ElMessage.error(error?.response?.data?.message || t('im.avatarUploadFailed'))
      } finally {
        avatarUploading.value = false
      }
      return false
    }
    async function selectSystemAvatar(avatar) {
      profile.avatar_url = avatar.value
      await api.put('/api/v1/accounts/me/profile', profilePayload())
      await refreshAvatarDisplay()
      syncCurrentUser()
      ElMessage.success(t('common.message.opSuccess'))
    }
    async function changePassword() {
      if (password.new_password !== password.confirm_password) {
        ElMessage.error(t('system.message.passwordMismatch'))
        return
      }
      await api.put('/api/v1/accounts/me/password', password)
      ElMessage.success(t('common.message.opSuccess'))
      localStorage.removeItem('token')
      localStorage.removeItem('current_user')
      window.location.href = '/'
      password.current_password = ''
      password.new_password = ''
      password.confirm_password = ''
      await load()
    }
    async function revokeOthers() {
      await api.delete('/api/v1/accounts/me/sessions')
      ElMessage.success(t('common.message.opSuccess'))
      await load()
    }
    function avatarText() {
      return (profile.name || profile.username || 'U').slice(0, 1).toUpperCase()
    }
    function profilePayload() {
      return {
        ...profile,
        department_id: Number(profile.department_id || 0),
        post_id: Number(profile.post_id || 0)
      }
    }
    function syncCurrentUser() {
      try {
        const current = JSON.parse(localStorage.getItem('current_user') || '{}')
        localStorage.setItem('current_user', JSON.stringify({ ...current, ...profile, avatar_url: profile.avatar_url }))
        window.dispatchEvent(new Event('current-user-updated'))
      } catch {
        // ignore cache sync failures
      }
    }
    function systemAvatarDataUrl(from, to, text) {
      const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="160" height="160" viewBox="0 0 160 160"><defs><linearGradient id="g" x1="0" x2="1" y1="0" y2="1"><stop stop-color="${from}"/><stop offset="1" stop-color="${to}"/></linearGradient></defs><rect width="160" height="160" rx="80" fill="url(#g)"/><circle cx="80" cy="64" r="30" fill="rgba(255,255,255,.88)"/><path d="M34 132c8-28 28-44 46-44s38 16 46 44" fill="rgba(255,255,255,.88)"/><text x="80" y="150" text-anchor="middle" font-family="Arial, sans-serif" font-size="26" font-weight="700" fill="rgba(255,255,255,.92)">${text}</text></svg>`
      return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`
    }
    onMounted(async () => {
      await load()
      await refreshAvatarDisplay()
    })
    watch(() => profile.avatar_url, refreshAvatarDisplay)
    onBeforeUnmount(() => revokeObjectUrl(avatarDisplayUrl.value))
    return () => h('div', { class: 'user-center-page' }, [
      h('section', { class: 'profile-card' }, [
        h('div', { class: 'profile-cover', style: profile.cover_url ? { backgroundImage: `url(${profile.cover_url})` } : undefined }),
        avatarDisplayUrl.value
          ? h('img', { class: 'profile-avatar image', src: avatarDisplayUrl.value, alt: t('domain.auto.4ceeeb31a5') })
          : h('div', { class: 'profile-avatar' }, avatarText()),
        h('h2', profile.name || profile.username || '-'),
        h('p', profile.username || '-'),
        h(resolveUpload(), {
          class: 'avatar-upload',
          showFileList: false,
          accept: 'image/jpeg,image/png,image/webp,image/gif',
          beforeUpload: uploadAvatar,
          disabled: avatarUploading.value
        }, () => h(resolveButton(), { size: 'small', type: 'primary', plain: true, loading: avatarUploading.value }, () => t('im.uploadAvatar'))),
        h('div', { class: 'system-avatar-panel' }, [
          h('strong', t('profile.avatar.systemTitle')),
          h('div', { class: 'system-avatar-grid' }, systemAvatarOptions.map((avatar) => h('button', {
            key: avatar.key,
            type: 'button',
            class: ['system-avatar-option', profile.avatar_url === avatar.value ? 'active' : ''],
            title: avatar.label,
            onClick: () => selectSystemAvatar(avatar)
          }, [h('img', { src: avatar.value, alt: avatar.label })])))
        ]),
        h('div', { class: 'profile-info' }, [
          h('span', `${t('domain.auto.783b0a5b8c')}${profile.email || '-'}`),
          h('span', `${t('domain.auto.8e187b3203')}${profile.post_name || t('domain.auto.1020bb8925')}`),
          h('span', `${t('domain.auto.e6d946c6ab')}${profile.address || t('domain.auto.8e9e5b046f')}`),
          h('span', `${t('domain.auto.45b54e2247')}${profile.department_name || t('domain.auto.c1507ee2d5')}`)
        ]),
        h('div', { class: 'profile-tags' }, [h('strong', t('domain.auto.ae0a7afece')), ...roleNames.value.map((name) => h('span', { class: 'role-chip' }, name))])
      ]),
      h('main', { class: 'profile-main' }, [
        h('section', { class: 'system-card profile-section' }, [
          h('div', { class: 'section-title' }, [h('h3', t('domain.auto.b68228a04e')), h('p', t('domain.auto.199612ead7'))]),
          h('div', { class: 'profile-grid' }, [
            field(t('domain.auto.25124ed74c'), h(resolveInput(), { modelValue: profile.username, 'onUpdate:modelValue': (v) => { profile.username = v }, disabled: true })),
            field(t('domain.auto.be4c2616b1'), h(resolveInput(), { modelValue: profile.name, 'onUpdate:modelValue': (v) => { profile.name = v } })),
            field(t('domain.auto.fe8aa4ef93'), h(resolveSelect(), { modelValue: profile.gender, 'onUpdate:modelValue': (v) => { profile.gender = v } }, () => [h(resolveOption(), { label: t('domain.auto.51625d909c'), value: t('domain.auto.51625d909c') }), h(resolveOption(), { label: t('domain.auto.26bc84961c'), value: t('domain.auto.26bc84961c') })])),
            field(t('domain.auto.9ed627bcf6'), h(resolveInput(), { modelValue: profile.email, 'onUpdate:modelValue': (v) => { profile.email = v } })),
            field(t('domain.auto.9c01ad0911'), h(resolveInput(), { modelValue: profile.phone, 'onUpdate:modelValue': (v) => { profile.phone = v } })),
            field(t('domain.auto.85ba8d2784'), h(resolveInput(), { modelValue: profile.address, 'onUpdate:modelValue': (v) => { profile.address = v } })),
            field(t('domain.auto.dc804dece5'), h(resolveInput(), { modelValue: profile.cover_url, 'onUpdate:modelValue': (v) => { profile.cover_url = v }, placeholder: t('domain.auto.0d8cbfa75e') })),
            field(t('domain.auto.5ff376c0bf'), h(resolveTreeSelect(), { modelValue: profile.department_id || undefined, 'onUpdate:modelValue': (v) => { profile.department_id = v }, data: mapTree(departments.value), checkStrictly: true, clearable: true, placeholder: t('domain.auto.50210083eb') })),
            field(t('domain.auto.552f0fe210'), h(resolveSelect(), { modelValue: profile.post_id || undefined, 'onUpdate:modelValue': (v) => { profile.post_id = v }, clearable: true, placeholder: t('domain.auto.dc5de8e845') }, () => posts.value.map((item) => h(resolveOption(), { key: item.id, label: item.name, value: item.id })))),
            h('label', { class: 'field wide' }, [h('span', t('domain.auto.eff7665554')), h(resolveInput(), { modelValue: profile.bio, 'onUpdate:modelValue': (v) => { profile.bio = v }, type: 'textarea', rows: 4, maxlength: 500, showWordLimit: true })])
          ]),
          h('div', { class: 'form-tail' }, [h(resolveButton(), { type: 'primary', onClick: saveProfile }, () => t('domain.auto.33f24b381b'))])
        ]),
        h('section', { class: 'system-card profile-section' }, [
          h('div', { class: 'section-title' }, [h('h3', t('domain.auto.7f01311f6c')), h('p', t('domain.auto.de11017fed'))]),
          h('div', { class: 'password-layout' }, [
            h('div', { class: 'profile-grid' }, [
              field(t('domain.auto.33abf70fd5'), h(resolveInput(), { modelValue: password.current_password, 'onUpdate:modelValue': (v) => { password.current_password = v }, type: 'password', showPassword: true })),
              field(t('domain.auto.d22c9c0085'), h(resolveInput(), { modelValue: password.new_password, 'onUpdate:modelValue': (v) => { password.new_password = v }, type: 'password', showPassword: true })),
              field(t('domain.auto.d4477adb6f'), h(resolveInput(), { modelValue: password.confirm_password, 'onUpdate:modelValue': (v) => { password.confirm_password = v }, type: 'password', showPassword: true }))
            ]),
            h('aside', { class: 'security-tips' }, [h('strong', t('domain.auto.8244b23c83')), h('p', t('domain.auto.c3aa361e63')), h('p', t('domain.auto.160e691347')), h('p', t('domain.auto.98c656acf2'))])
          ]),
          h('div', { class: 'form-tail' }, [h(resolveButton(), { type: 'primary', onClick: changePassword }, () => t('domain.auto.2f3ed4c7a4'))])
        ]),
        h('section', { class: 'system-card profile-section' }, [
          h('div', { class: 'section-title row' }, [h('div', [h('h3', t('domain.auto.673610a5a3')), h('p', `${t('domain.auto.3b6ef811b8')}${sessionPage.total}${t('domain.auto.b5497e6733')}`)]), h(resolveButton(), { type: 'danger', plain: true, onClick: revokeOthers }, () => t('domain.auto.3133495fa2'))]),
          h('div', { class: 'session-list' }, sessions.value.map((item) => h('div', { class: 'session-item' }, [
            h('span', { class: 'session-icon' }, [h(Icon, { icon: 'ri:computer-line' })]),
            h('div', [h('strong', `${item.os || t('domain.auto.d9c32a4c3d')}${t('domain.auto.01f2c16cda')}`), h('p', `${item.browser || t('domain.auto.d9c32a4c3d')}　IP：${item.ip || '-'}${t('domain.auto.230c92c6d2')}${formatTime(item.last_active)}`), h('p', item.session_id === currentSessionId.value ? t('domain.auto.ea680f8349') : (item.revoked ? t('domain.auto.0456cdca6b') : t('domain.auto.0373ff9231')))]),
            h('span', { class: ['status-pill', item.revoked ? 'off' : 'on'] }, item.revoked ? t('domain.auto.0456cdca6b') : t('domain.auto.0373ff9231'))
          ]))),
          h('div', { class: 'system-pagination session-pagination' }, [
            h(resolvePagination(), {
              currentPage: sessionPage.page,
              'onUpdate:currentPage': (value) => { sessionPage.page = value },
              pageSize: sessionPage.size,
              'onUpdate:pageSize': (value) => { sessionPage.size = value },
              layout: 'prev, pager, next',
              total: sessionPage.total,
              onCurrentChange: load
            })
          ])
        ])
      ])
    ])
  }
})

function resolveInput() { return resolveComponent('ElInput') }
function resolveSelect() { return resolveComponent('ElSelect') }
function resolveTreeSelect() { return resolveComponent('ElTreeSelect') }
function resolveOption() { return resolveComponent('ElOption') }
function resolveButton() { return resolveComponent('ElButton') }
function resolvePagination() { return resolveComponent('ElPagination') }
function resolveUpload() { return resolveComponent('ElUpload') }
function field(label, node) { return h('label', { class: 'field' }, [h('span', label), node]) }

const route = useRoute()
const section = computed(() => route.meta.systemSection || 'user')
const searchableSections = ['user', 'role', 'log', 'login-log', 'security-log']
const loading = ref(false)
const searchExpanded = ref(false)
const expandAll = ref(false)
const users = ref([])
const roles = ref([])
const departments = ref([])
const posts = ref([])
const menus = ref([])
const logs = ref([])
const formRef = ref(null)
const menuFormRef = ref(null)
const tableCardRef = ref(null)
const menuDrawer = ref(false)
const columnDrawer = ref(false)
const isFullscreen = ref(false)
const dialog = reactive({ visible: false, title: '', mode: 'create', id: null })
const resetPasswordDialog = reactive({ visible: false, saving: false, userId: 0, username: '', password: '' })
const permissionDrawer = reactive({ visible: false, role: null, selected: [], saving: false })
const filters = reactive({})
const form = reactive({})
const pagination = reactive({ page: 1, size: 20, total: 0 })
const columnOptions = {
  user: [
    { key: 'username', label: t('domain.auto.a1aaf352cb') },
    { key: 'name', label: t('domain.auto.be4c2616b1') },
    { key: 'gender', label: t('domain.auto.fe8aa4ef93') },
    { key: 'phone', label: t('domain.auto.5a9cc5e891') },
    { key: 'roles', label: t('domain.auto.e323915cc2') },
    { key: 'department', label: t('domain.auto.5ff376c0bf') },
    { key: 'post', label: t('domain.auto.552f0fe210') },
    { key: 'created_at', label: t('domain.auto.6102c5c9e6') },
    { key: 'updated_at', label: t('domain.auto.beef2c086d') }
  ],
  role: [
    { key: 'id', label: t('domain.auto.c38a312709') },
    { key: 'name', label: t('domain.auto.3aa1f085b2') },
    { key: 'code', label: t('domain.auto.c12ace673d') },
    { key: 'description', label: t('domain.auto.a0a62a0031') },
    { key: 'status', label: t('domain.auto.c665191612') },
    { key: 'created_at', label: t('domain.auto.6102c5c9e6') },
    { key: 'updated_at', label: t('domain.auto.9b9857e025') }
  ],
  department: [
    { key: 'name', label: t('domain.auto.a35c1d2eb8') },
    { key: 'code', label: t('domain.auto.e43f2be96c') },
    { key: 'leader', label: t('domain.auto.974d383f36') },
    { key: 'phone', label: t('domain.auto.e02f6e5760') },
    { key: 'email', label: t('domain.auto.9ed627bcf6') },
    { key: 'sort', label: t('domain.auto.dc35af8d69') },
    { key: 'status', label: t('domain.auto.62e951a692') },
    { key: 'updated_at', label: t('domain.auto.093dea88c9') }
  ],
  post: [
    { key: 'name', label: t('domain.auto.f7e2b452ba') },
    { key: 'code', label: t('domain.auto.0fc3644a4f') },
    { key: 'sort', label: t('domain.auto.dc35af8d69') },
    { key: 'status', label: t('domain.auto.62e951a692') },
    { key: 'remark', label: t('domain.auto.e0361480e3') },
    { key: 'updated_at', label: t('domain.auto.093dea88c9') }
  ],
  menu: [
    { key: 'name', label: t('domain.auto.2637dede68') },
    { key: 'type', label: t('domain.auto.7907c3c545') },
    { key: 'path', label: t('domain.auto.19da0c94ef') },
    { key: 'permission', label: t('domain.auto.18a0035e57') },
    { key: 'sort', label: t('domain.auto.dc35af8d69') },
    { key: 'updated_at', label: t('domain.auto.0f92576ac5') },
    { key: 'status', label: t('domain.auto.62e951a692') }
  ]
}
const columnSections = Object.keys(columnOptions)
const sortableSections = ['user', 'role', 'department', 'post', 'menu', 'log', 'login-log', 'security-log']
const sortDefaults = {
  user: 'created_at',
  role: 'created_at',
  department: 'sort',
  post: 'sort',
  menu: 'sort',
  log: 'created_at',
  'login-log': 'created_at',
  'security-log': 'created_at'
}
const sortState = reactive({ user: 'desc', role: 'desc', department: 'asc', post: 'asc', menu: 'asc', log: 'desc', 'login-log': 'desc', 'security-log': 'desc' })
const columnSelections = reactive(Object.fromEntries(columnSections.map((key) => [key, loadStoredColumns(key)])))

const createButtonText = computed(() => ({
  user: t('system.button.createUser'),
  role: t('system.button.createRole'),
  department: t('system.button.createDepartment'),
  post: t('system.button.createPost'),
  menu: t('system.button.createMenu')
}[section.value] || t('common.action.create')))
const dialogWidth = computed(() => {
  if (section.value === 'role') return '760px'
  if (section.value === 'department') return '720px'
  return '540px'
})
const departmentSelectTree = computed(() => mapTree(departments.value))
const menuSelectTree = computed(() => mapTree(menus.value))
const flatDepartments = computed(() => flattenTree(departments.value))
const currentColumnOptions = computed(() => columnOptions[section.value] || [])
const currentSortDirection = computed(() => sortState[section.value] || 'asc')
const sortTooltip = computed(() => t('system.tooltip.sortBy').replace('{field}', sortLabel(section.value)).replace('{order}', currentSortDirection.value === 'asc' ? t('system.sort.asc') : t('system.sort.desc')))
const sortIcon = computed(() => currentSortDirection.value === 'asc' ? 'ri:sort-asc' : 'ri:sort-desc')
const columnSignature = computed(() => `${section.value}:${(columnSelections[section.value] || []).join('|')}:${currentSortDirection.value}:${expandAll.value}`)
const userRows = computed(() => sortFlatRows(users.value, section.value))
const roleRows = computed(() => sortFlatRows(roles.value, section.value))
const postRows = computed(() => sortFlatRows(posts.value, section.value))
const departmentRows = computed(() => sortTreeRows(departments.value, section.value))
const menuRows = computed(() => sortTreeRows(menus.value, section.value))
const logRows = computed(() => sortFlatRows(logs.value, section.value))
const isLogSection = computed(() => ['log', 'login-log', 'security-log'].includes(section.value))
const logRowPrefix = computed(() => ({ 'login-log': 'LG', 'security-log': 'SEC' }[section.value] || 'OP'))
const userLookup = computed(() => new Map((users.value || []).map((item) => [Number(item.id), item])))
const roleLookup = computed(() => new Map((roles.value || []).map((item) => [Number(item.id), item])))
const permissionDrawerTitle = computed(() => t('system.permission.drawerTitle', { role: roleDisplayName(permissionDrawer.role) }))
const leaderOptions = computed(() => users.value.map((item) => ({
  value: item.name || item.username || item.email || `User${item.id}`,
  label: `${item.name || item.username || item.email || `User${item.id}`}${item.username ? ` (${item.username})` : ''}${item.department_name ? ` · ${item.department_name}` : ''}`
})))
const rules = computed(() => ({
  username: [{ required: true, message: t('domain.auto.ad5f05f1cf'), trigger: 'blur' }],
  password: [{ required: dialog.mode === 'create', min: 6, message: t('domain.auto.9bfe0ae551'), trigger: 'blur' }],
  name: [{ required: ['role', 'department', 'post', 'menu'].includes(section.value), message: t('domain.auto.c2afb255a5'), trigger: 'blur' }],
  code: [{ required: ['department', 'post'].includes(section.value), message: t('domain.auto.76745b527c'), trigger: 'blur' }],
  role_ids: [{ required: section.value === 'user', type: 'array', min: 1, message: t('domain.auto.30741f3cf7'), trigger: 'change' }]
}))

const rolePresetMap = {
  salesperson: { name: t('system.rolePreset.salesperson'), code: 'R_SALES', description: t('system.rolePreset.salespersonDesc') },
  supervisor: { name: t('domain.auto.f7751161e4'), code: 'R_SUPERVISOR', description: t('domain.auto.e521198125') },
  sdr_supervisor: { name: t('domain.auto.396d9b8c30'), code: 'R_SDR_SUPERVISOR', description: t('domain.auto.9dfa5da7a5') }
}
const backendRoleSeeds = {
  admin: { name: '\u7ba1\u7406\u5458', description: '\u4f01\u4e1a\u7ba1\u7406\u5458\u9884\u8bbe\u89d2\u8272' },
  editor: { name: '\u7f16\u8f91\u8005', description: '\u53ef\u7f16\u8f91\u4e1a\u52a1\u6570\u636e\u7684\u9884\u8bbe\u89d2\u8272' },
  readonly: { name: '\u53ea\u8bfb\u7528\u6237', description: '\u53ea\u8bfb\u8bbf\u95ee\u4e1a\u52a1\u6a21\u5757\u7684\u9884\u8bbe\u89d2\u8272' }
}
const permissionCatalogRows = ref([])
const enabledModuleSet = computed(() => {
  const set = new Set((readEnabledModules() || []).map((item) => String(item || '').trim()).filter(Boolean))
  // Audit logs are mandatory tenant capabilities by policy.
  set.add('audit_log')
  set.add('enterprise')
  set.add('storage')
  set.add('iot')
  set.add('accounts')
  return set
})
const resourceModuleMap = {
  'system:department': 'accounts',
  'system:post': 'accounts'
}
const businessPermissionResourceOrder = [
  'workspace',
  'dashboards',
  'im',
  'campaigns',
  'leads',
  'contacts',
  'opportunities',
  'deals',
  'orders',
  'tickets',
  'projects',
  'products',
  'insights',
  'mfg',
  'mfg:rfq',
  'mfg:product',
  'mfg:cost',
  'mfg:quote',
  'mfg:sample',
  'mfg:production',
  'mfg:ecn',
  'mfg:metrics'
]
const platformPermissionResourceOrder = ['enterprise', 'accounts', 'system:department', 'system:post', 'storage', 'iot', 'audit_log']
const visiblePermissionResourceSet = new Set([...businessPermissionResourceOrder, ...platformPermissionResourceOrder])
const businessPermissionOrderMap = new Map(businessPermissionResourceOrder.map((resource, index) => [resource, index]))
const platformPermissionOrderMap = new Map(platformPermissionResourceOrder.map((resource, index) => [resource, index]))
function permissionResourceVisible(resource) {
  if (!visiblePermissionResourceSet.has(resource)) return false
  const mapped = resourceModuleMap[resource] || resource
  return enabledModuleSet.value.has(mapped)
}
function permissionResourceOrder(resource, orderMap) {
  return orderMap.has(resource) ? orderMap.get(resource) : 999
}
const defaultFineActions = ['view', 'create', 'update', 'delete', 'export', 'manage', 'view_sensitive']
const businessPermissionOptions = computed(() => (permissionCatalogRows.value || [])
  .filter((item) => item.group !== 'platform' && permissionResourceVisible(item.resource))
  .map((item) => ({
    ...item,
    hint: permissionResourceHint(item.resource, item.label),
    actions: (item.actions || []).filter((action) => defaultFineActions.includes(action) || action === 'assign' || action === 'approve')
  }))
  .sort((a, b) => permissionResourceOrder(a.resource, businessPermissionOrderMap) - permissionResourceOrder(b.resource, businessPermissionOrderMap)))
const platformPermissionOptions = computed(() => (permissionCatalogRows.value || [])
  .filter((item) => item.group === 'platform' && permissionResourceVisible(item.resource))
  .map((item) => ({
    ...item,
    hint: permissionResourceHint(item.resource, item.label),
    actions: (item.actions || []).filter((action) => defaultFineActions.includes(action) || action === 'assign' || action === 'approve')
  }))
  .sort((a, b) => permissionResourceOrder(a.resource, platformPermissionOrderMap) - permissionResourceOrder(b.resource, platformPermissionOrderMap)))
const availablePermissionSet = computed(() => {
  const set = new Set()
  for (const item of businessPermissionOptions.value) {
    for (const action of item.actions || []) {
      set.add(`${item.resource}:${action}`)
    }
  }
  for (const item of platformPermissionOptions.value) {
    for (const action of item.actions || []) {
      set.add(`${item.resource}:${action}`)
    }
  }
  return set
})

watch(section, async () => {
  if (section.value === 'user-center') return
  isFullscreen.value = false
  columnDrawer.value = false
  resetFilters()
  resetForm()
  await loadLookups()
  await loadPage()
}, { immediate: true })

async function loadLookups() {
  const [rolesRes, deptRes, postRes, menuRes, usersRes, permissionCatalogRes] = await Promise.allSettled([
    hasPermission('accounts:manage') ? api.get('/api/v1/rbac/roles') : Promise.resolve({ data: { items: [] } }),
    hasPermission('system:department:view') ? api.get('/api/v1/system/departments') : Promise.resolve({ data: { items: [] } }),
    hasPermission('system:post:view') ? api.get('/api/v1/system/posts', { params: { page: 1, size: 200 } }) : Promise.resolve({ data: { items: [] } }),
    hasPermission('system:menu:view') ? api.get('/api/v1/system/menus', { params: { visible: 1 } }) : Promise.resolve({ data: { items: [] } }),
    hasPermission('accounts:manage') ? api.get('/api/v1/accounts/users', { params: { page: 1, size: 500 } }) : Promise.resolve({ data: { items: [] } }),
    hasPermission('accounts:manage') ? api.get('/api/v1/rbac/permissions/catalog') : Promise.resolve({ data: { items: [] } })
  ])
  if (rolesRes.status === 'fulfilled') roles.value = rolesRes.value.data?.items || []
  if (deptRes.status === 'fulfilled') departments.value = deptRes.value.data?.items || []
  if (postRes.status === 'fulfilled') posts.value = postRes.value.data?.items || []
  if (menuRes.status === 'fulfilled') menus.value = menuRes.value.data?.items || []
  if (usersRes.status === 'fulfilled') users.value = usersRes.value.data?.items || []
  if (permissionCatalogRes.status === 'fulfilled') {
    permissionCatalogRows.value = (permissionCatalogRes.value.data?.items || []).map((item) => {
      const group = platformPermissionOrderMap.has(item.resource) ? 'platform' : 'business'
      return {
        resource: item.resource,
        label: permissionLabel(item.resource),
        actions: item.actions || [],
        group
      }
    })
  } else {
    permissionCatalogRows.value = []
  }
}

async function loadPage() {
  loading.value = true
  try {
    if (section.value === 'user') await loadUsers()
    else if (section.value === 'role') await loadRoles()
    else if (section.value === 'department') await loadDepartments()
    else if (section.value === 'post') await loadPosts()
    else if (section.value === 'menu') await loadMenus()
    else await loadLogs()
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error.message || t('common.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadUsers() {
  const params = cleanParams({ ...filters, page: pagination.page, size: pagination.size, start_time: filters.date_range?.[0], end_time: filters.date_range?.[1] })
  delete params.date_range
  const res = await api.get('/api/v1/accounts/users', { params })
  users.value = res.data?.items || []
  pagination.total = res.data?.total || 0
}
async function loadRoles() {
  const params = cleanParams({ ...filters, start_time: filters.date_range?.[0], end_time: filters.date_range?.[1] })
  delete params.date_range
  const res = await api.get('/api/v1/rbac/roles', { params })
  roles.value = res.data?.items || []
  pagination.total = res.data?.total || roles.value.length
}
async function loadDepartments() {
  const res = await api.get('/api/v1/system/departments')
  departments.value = res.data?.items || []
}
async function loadPosts() {
  const res = await api.get('/api/v1/system/posts', { params: { page: pagination.page, size: pagination.size } })
  posts.value = res.data?.items || []
  pagination.total = res.data?.total || 0
}
async function loadMenus() {
  const res = await api.get('/api/v1/system/menus', { params: { visible: 1 } })
  menus.value = res.data?.items || []
}
async function loadLogs() {
  const params = cleanParams({ user: filters.user, result: filters.result, operation: logOperationFilter(), start_time: filters.date_range?.[0], end_time: filters.date_range?.[1], page: pagination.page, size: pagination.size })
  const res = await api.get('/api/v1/operation-logs', { params })
  logs.value = res.data?.items || []
  pagination.total = res.data?.total || 0
}

function logOperationFilter() {
  if (section.value === 'login-log') return 'LOGIN'
  if (section.value === 'security-log') return 'SECURITY_*'
  return filters.operation
}

function openCreate(parent) {
  resetForm()
  dialog.mode = 'create'
  dialog.id = null
  dialog.title = createButtonText.value
  if (parent?.id) form.parent_id = parent.id
  if (section.value === 'menu') menuDrawer.value = true
  else dialog.visible = true
}

function openEdit(row) {
  resetForm()
  Object.assign(form, row)
  form.enabled = row.status !== 'disabled'
  form.role_ids = row.role_ids || (row.role_id ? [row.role_id] : [])
  if (section.value === 'role') form.role_preset = inferRolePreset(row)
  dialog.mode = 'edit'
  dialog.id = row.id
  dialog.title = createButtonText.value.replace(t('common.action.create'), t('common.action.edit'))
  if (section.value === 'menu') menuDrawer.value = true
  else dialog.visible = true
}

async function submitForm() {
  const ref = section.value === 'menu' ? menuFormRef.value : formRef.value
  await ref?.validate()
  const payload = { ...form, status: form.enabled ? 'active' : 'disabled' }
  if (section.value === 'user') {
    if (dialog.mode === 'create') await api.post('/api/v1/accounts/users', payload)
    else await api.put(`/api/v1/accounts/users/${dialog.id}`, payload)
  } else if (section.value === 'role') {
    if (dialog.mode === 'create') await api.post('/api/v1/rbac/roles', payload)
    else await api.put(`/api/v1/rbac/roles/${dialog.id}`, payload)
  } else if (section.value === 'department') {
    if (dialog.mode === 'create') await api.post('/api/v1/system/departments', payload)
    else await api.put(`/api/v1/system/departments/${dialog.id}`, payload)
  } else if (section.value === 'post') {
    if (dialog.mode === 'create') await api.post('/api/v1/system/posts', payload)
    else await api.put(`/api/v1/system/posts/${dialog.id}`, payload)
  } else if (section.value === 'menu') {
    if (dialog.mode === 'create') await api.post('/api/v1/system/menus', payload)
    else await api.put(`/api/v1/system/menus/${dialog.id}`, payload)
  }
  ElMessage.success(t('common.message.opSuccess'))
  dialog.visible = false
  menuDrawer.value = false
  await loadLookups()
  await loadPage()
}

async function openResetPassword(row) {
  if (row?.is_primary_admin) {
    ElMessage.warning(t('system.user.primaryAdminProtected'))
    return
  }
  resetPasswordDialog.userId = Number(row?.id || 0)
  resetPasswordDialog.username = row?.username || ''
  resetPasswordDialog.password = ''
  resetPasswordDialog.visible = true
}

async function submitResetPassword() {
  if (!resetPasswordDialog.userId || resetPasswordDialog.saving) return
  if (String(resetPasswordDialog.password || '').trim().length < 6) {
    ElMessage.error(t('system.user.passwordRule'))
    return
  }
  resetPasswordDialog.saving = true
  try {
    await api.put(`/api/v1/accounts/users/${resetPasswordDialog.userId}/password`, { password: resetPasswordDialog.password })
    ElMessage.success(t('system.user.passwordResetSuccess'))
    resetPasswordDialog.visible = false
    resetPasswordDialog.password = ''
  } catch (error) {
    ElMessage.error(localizedSystemError(error, t('system.user.passwordResetFailed')))
  } finally {
    resetPasswordDialog.saving = false
  }
}

async function toggleUserStatus(row) {
  if (row?.is_primary_admin) {
    ElMessage.warning(t('system.user.primaryAdminProtected'))
    return
  }
  const nextStatus = row.status === 'disabled' ? 'active' : 'disabled'
  await api.put(`/api/v1/accounts/users/${row.id}`, { ...row, status: nextStatus })
  ElMessage.success(t('common.message.opSuccess'))
  await loadPage()
}

const deleteUser = (id) => api.delete(`/api/v1/accounts/users/${id}`).then(doneDelete).catch((error) => {
  ElMessage.error(localizedSystemError(error, t('common.message.loadFailed')))
})
const deleteRole = (id) => api.delete(`/api/v1/rbac/roles/${id}`).then(doneDelete)
const deleteDepartment = (id) => api.delete(`/api/v1/system/departments/${id}`).then(doneDelete)
const deletePost = (id) => api.delete(`/api/v1/system/posts/${id}`).then(doneDelete)
const deleteMenu = (id) => api.delete(`/api/v1/system/menus/${id}`).then(doneDelete)

async function doneDelete() {
  ElMessage.success(t('common.message.opSuccess'))
  await loadLookups()
  await loadPage()
}

async function toggleRole(row) {
  await api.put(`/api/v1/rbac/roles/${row.id}`, { ...row, status: row.status === 'disabled' ? 'active' : 'disabled' })
  ElMessage.success(t('common.message.opSuccess'))
  await loadRoles()
}

function openPermissionDrawer(row) {
  permissionDrawer.role = row
  permissionDrawer.selected = isAdminPresetRole(row)
    ? [...availablePermissionSet.value]
    : splitPermissions(row.permissions_json).filter((item) => availablePermissionSet.value.has(item))
  permissionDrawer.visible = true
}

async function saveRolePermissions() {
  if (!permissionDrawer.role?.id || permissionDrawer.saving) return
  permissionDrawer.saving = true
  try {
    const visiblePermissions = isAdminPresetRole(permissionDrawer.role)
      ? [...availablePermissionSet.value]
      : permissionDrawer.selected.filter((item) => availablePermissionSet.value.has(item))
    const hiddenExistingPermissions = splitPermissions(permissionDrawer.role.permissions_json).filter((item) => !availablePermissionSet.value.has(item))
    const grouped = groupPermissions([...hiddenExistingPermissions, ...visiblePermissions])
    const res = await api.post('/api/v1/rbac/role-permissions', {
      role_id: permissionDrawer.role.id,
      permissions: grouped,
      data_scope: 'custom'
    })
    const nextPermissions = res.data?.permissions_json || permissionDrawer.selected.join(',')
    permissionDrawer.role.permissions_json = nextPermissions
    const role = roles.value.find((item) => Number(item.id) === Number(permissionDrawer.role.id))
    if (role) role.permissions_json = nextPermissions
    ElMessage.success(t('system.message.permissionSaved'))
    permissionDrawer.visible = false
    await loadRoles()
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('system.permission.saveFailed'))
  } finally {
    permissionDrawer.saving = false
  }
}

function selectAllPermissions() {
  permissionDrawer.selected = [...availablePermissionSet.value]
}

function clearAllPermissions() {
  permissionDrawer.selected = []
}

function splitPermissions(value = '') {
  return String(value).split(',').map((item) => item.trim()).filter(Boolean)
}

function groupPermissions(items = []) {
  const map = new Map()
  for (const item of items) {
    const index = item.lastIndexOf(':')
    if (index <= 0) continue
    const resource = item.slice(0, index)
    const action = item.slice(index + 1)
    if (!map.has(resource)) map.set(resource, new Set())
    map.get(resource).add(action)
  }
  return [...map.entries()].map(([resource, actions]) => ({ resource, actions: [...actions] }))
}

function confirmDelete(type, name, action) {
  ElMessageBox.confirm(t('system.message.confirmDelete').replace('{name}', name), t('system.message.deleteTitle').replace('{type}', type), { type: 'warning', confirmButtonText: t('common.action.submit'), cancelButtonText: t('common.action.cancel') }).then(action).catch(() => {})
}

function localizedSystemError(error, fallback) {
  const body = error?.response?.data || {}
  const code = body?.data?.error_code || body?.error_code || body?.message
  if (code === 'PRIMARY_TENANT_ADMIN_PROTECTED') return t('system.user.primaryAdminProtected')
  return apiErrorMessage(error, fallback)
}

function resetFilters() {
  Object.keys(filters).forEach((key) => delete filters[key])
  pagination.page = 1
}
function resetForm() {
  Object.keys(form).forEach((key) => delete form[key])
  Object.assign(form, { sort: 1, enabled: true, status: 'active', gender: 'male', type: 'catalog', role_ids: [], role_preset: 'normal' })
}
function applyRolePreset(value) {
  if (value === 'normal') {
    form.name = ''
    form.code = ''
    form.description = ''
    return
  }
  const preset = rolePresetMap[value]
  if (!preset) return
  form.name = preset.name
  form.code = preset.code
  form.description = preset.description
}
function inferRolePreset(role = {}) {
  const text = `${role.name || ''} ${role.code || ''}`.toLowerCase()
  if (text.includes('销售员') || text.includes('销售代表') || text.includes('r_sales') || text.includes('salesperson') || text.includes('sales_rep')) return 'salesperson'
  if (text.includes('sdr') && (text.includes(t('domain.auto.f7751161e4')) || text.includes('supervisor') || text.includes('manager'))) return 'sdr_supervisor'
  if (text.includes(t('domain.auto.f7751161e4')) || text.includes('supervisor') || text.includes('manager')) return 'supervisor'
  return 'normal'
}
function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
  requestAnimationFrame(() => {
    tableCardRef.value?.scrollTo?.({ top: 0, behavior: 'smooth' })
  })
}
function openColumnSettings() {
  columnDrawer.value = true
}
function visibleColumn(key) {
  return (columnSelections[section.value] || []).includes(key)
}
function selectAllColumns() {
  columnSelections[section.value] = defaultColumns(section.value)
  saveColumnSettings()
}
function resetColumns() {
  columnSelections[section.value] = defaultColumns(section.value)
  localStorage.removeItem(columnStorageKey(section.value))
}
function saveColumnSettings() {
  if (!columnSelections[section.value]?.length) {
    columnSelections[section.value] = defaultColumns(section.value)
  }
  localStorage.setItem(columnStorageKey(section.value), JSON.stringify(columnSelections[section.value]))
}
function loadStoredColumns(key) {
  try {
    const allowed = defaultColumns(key)
    const stored = JSON.parse(localStorage.getItem(columnStorageKey(key)) || '[]')
    const valid = stored.filter((item) => allowed.includes(item))
    return valid.length ? valid : allowed
  } catch {
    return defaultColumns(key)
  }
}
function defaultColumns(key) {
  return (columnOptions[key] || []).map((item) => item.key)
}
function columnStorageKey(key) {
  return `system_columns_${key}`
}
function toggleSort() {
  if (!sortableSections.includes(section.value)) return
  sortState[section.value] = currentSortDirection.value === 'asc' ? 'desc' : 'asc'
}
function sortLabel(key) {
  return ({ user: t('domain.auto.6102c5c9e6'), role: t('domain.auto.6102c5c9e6'), department: t('domain.auto.dc35af8d69'), post: t('domain.auto.dc35af8d69'), menu: t('domain.auto.dc35af8d69'), log: t('domain.auto.219b76faca'), 'login-log': t('domain.auto.219b76faca'), 'security-log': t('domain.auto.219b76faca') }[key] || t('domain.auto.e40c64f840'))
}
function actionLabel(action) {
  return ({
    view: t('domain.auto.f7acefd2d4'),
    create: t('domain.auto.2cd9e6ce81'),
    update: t('domain.auto.a7f814c0a4'),
    delete: t('domain.auto.3755f56f2f'),
    export: t('domain.auto.188896795f'),
    view_sensitive: t('industry.manufacturing.permission.viewSensitive'),
    assign: t('domain.auto.bdddf08727'),
    approve: t('domain.auto.5ce60cb75d'),
    manage: t('domain.auto.4989b5cf94')
  }[action] || action)
}
function permissionResourceHint(resource, label) {
  if (resource === 'dashboards') return t('system.permission.hint.dashboards')
  if (resource === 'teams') return t('system.permission.hint.teams')
  if (resource === 'contacts') return t('system.permission.hint.contacts')
  if (resource === 'enterprise') return t('system.permission.hint.enterprise')
  if (resource === 'accounts') return t('system.permission.hint.accounts')
  if (resource === 'system:department') return t('system.permission.hint.department')
  if (resource === 'system:post') return t('system.permission.hint.post')
  if (resource === 'storage') return t('system.permission.hint.storage')
  if (resource === 'iot') return t('system.permission.hint.iot')
  if (resource === 'audit_log') return t('system.permission.hint.auditLog')
  return t('system.permission.hint.default', { label })
}
function permissionAllowText(action, label) {
  return ({
    view: t('system.permission.allow.view', { label }),
    create: t('system.permission.allow.create', { label }),
    update: t('system.permission.allow.update', { label }),
    delete: t('system.permission.allow.delete', { label }),
    export: t('system.permission.allow.export', { label }),
    assign: t('system.permission.allow.assign', { label }),
    approve: t('system.permission.allow.approve', { label }),
    manage: t('system.permission.allow.manage', { label })
  }[action] || t('system.permission.allow.default', { action: actionLabel(action) }))
}
function permissionDenyText(action, label) {
  return ({
    view: t('system.permission.deny.view'),
    create: t('system.permission.deny.create', { label }),
    update: t('system.permission.deny.update', { label }),
    delete: t('system.permission.deny.delete', { label }),
    export: t('system.permission.deny.export'),
    assign: t('system.permission.deny.assign', { label }),
    approve: t('system.permission.deny.approve'),
    manage: t('system.permission.deny.manage')
  }[action] || t('system.permission.deny.default', { action: actionLabel(action) }))
}
function permissionTooltipText(action, label) {
  return `${permissionAllowText(action, label)} ${permissionDenyText(action, label)}`
}
function permissionLabel(resource) {
  return ({
    workspace: t('domain.auto.dd3edc066c'),
    projects: t('domain.auto.6dd16d11ee'),
    products: t('domain.auto.76363b4539'),
    campaigns: t('domain.auto.5fa95b1b73'),
    contacts: t('system.permission.contactsCombined'),
    leads: t('domain.auto.8de0b8fb1d'),
    opportunities: t('domain.auto.2559c4e3c0'),
    deals: t('domain.auto.6a30a471cf'),
    orders: t('domain.auto.81b3798beb'),
    tickets: t('domain.auto.17e2f8d878'),
    dashboards: t('domain.auto.e164aa4223'),
    teams: t('domain.auto.b060bbbc90'),
    insights: t('domain.auto.47b27c79bd'),
    enterprise: t('system.permission.resource.enterprise'),
    accounts: t('domain.auto.baf84751a2'),
    storage: t('system.permission.resource.storage'),
    iot: t('system.permission.resource.iot'),
    im: t('domain.auto.925f26b9e1'),
    mfg: t('industry.manufacturing.menu.title'),
    'mfg:rfq': t('industry.manufacturing.workspace.rfq'),
    'mfg:product': t('industry.manufacturing.workspace.product'),
    'mfg:cost': t('industry.manufacturing.workspace.cost'),
    'mfg:quote': t('industry.manufacturing.workspace.quote'),
    'mfg:sample': t('industry.manufacturing.workspace.sample'),
    'mfg:production': t('industry.manufacturing.workspace.production'),
    'mfg:ecn': t('industry.manufacturing.workspace.ecn'),
    'mfg:metrics': t('system.permission.resource.mfgMetrics'),
    'system:department': t('domain.auto.eca7562269'),
    'system:post': t('domain.auto.87fc305b3b'),
    audit_log: t('domain.auto.1db2e8417b')
  }[resource] || resource)
}
function sortFlatRows(items, key) {
  return [...(items || [])].sort((a, b) => compareRows(a, b, key))
}
function sortTreeRows(items, key) {
  return sortFlatRows(items, key).map((item) => ({ ...item, children: sortTreeRows(item.children || [], key) }))
}
function compareRows(a, b, key) {
  const direction = currentSortDirection.value === 'asc' ? 1 : -1
  const field = sortDefaults[key] || 'sort'
  const left = comparableValue(a?.[field])
  const right = comparableValue(b?.[field])
  if (left > right) return direction
  if (left < right) return -direction
  return Number(a?.id || 0) - Number(b?.id || 0)
}
function comparableValue(value) {
  if (value === undefined || value === null || value === '') return 0
  if (typeof value === 'number') return value
  const time = new Date(value).getTime()
  if (!Number.isNaN(time)) return time
  return String(value)
}
async function exportLogs() {
  const url = section.value === 'login-log' ? '/api/v1/operation-logs/login/export' : '/api/v1/operation-logs/export'
  const params = section.value === 'security-log' ? { operation: 'SECURITY_*' } : undefined
  const res = await api.get(url, { params, responseType: 'blob' })
  const blob = new Blob([res], { type: 'application/zip' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = ({ 'login-log': 'login_logs_excel.zip', 'security-log': 'security_logs_excel.zip' }[section.value] || 'operation_logs_excel.zip')
  document.body.appendChild(link)
  link.click()
  URL.revokeObjectURL(link.href)
  link.remove()
}
function showLog(row) {
  ElMessageBox.alert(renderLogDetail(row), t('domain.auto.88b864bdf6'), {
    dangerouslyUseHTMLString: true,
    confirmButtonText: t('domain.auto.f526c89937'),
    customClass: 'system-log-detail-message'
  })
}
function renderLogDetail(row) {
  const rows = logDetailRows(row)
  const body = rows.map((item) => `
    <div class="system-log-detail__row">
      <span>${escapeHtml(item.label)}</span>
      <strong>${escapeHtml(item.value)}</strong>
    </div>
  `).join('')
  return `<div class="system-log-detail">${body}</div>`
}
function logDetailRows(row) {
  const params = parseLogParams(row)
  const items = [
    { label: t('domain.auto.8cac83c83b'), value: `${logRowPrefix.value}${row?.id || ''}` },
    { label: t('domain.auto.219b76faca'), value: formatTime(row?.created_at) },
    { label: t('domain.auto.ae0849ff19'), value: row?.username || row?.user || (row?.user_id ? `#${row.user_id}` : '') },
    { label: t('domain.auto.c1bebd4ab3'), value: logModuleLabel(row) },
    { label: t('domain.auto.19e41f1bea'), value: operationLabel(row?.operation) },
    { label: t('domain.auto.539cb3b0a6'), value: logTargetLabel(row) },
    { label: t('domain.auto.62e951a692'), value: row?.result === 'failed' ? t('common.status.failed') : t('common.status.success') },
    { label: t('domain.auto.67d2d7970f'), value: row?.ip || params.ip },
    { label: t('domain.auto.01f2c16cda'), value: logDeviceLabel(row) },
    { label: t('domain.auto.26670dda42'), value: logDescription(row) }
  ]
  const extra = logReadableParams(params)
  if (extra) items.push({ label: t('domain.auto.4f55ee1e68'), value: extra })
  return items.filter((item) => item.value !== undefined && item.value !== null && String(item.value).trim() !== '' && String(item.value).trim() !== '-')
}
function logDeviceLabel(row) {
  const ua = String(row?.user_agent || '').trim()
  if (!ua) return ''
  return `${osFromUA(ua)} / ${browserFromUA(ua)}`
}
function logReadableParams(params = {}) {
  const hidden = new Set(['password', 'token', 'authorization', 'secret', 'credential'])
  return Object.entries(params)
    .filter(([key, value]) => value !== undefined && value !== null && value !== '' && !hidden.has(String(key).toLowerCase()))
    .slice(0, 6)
    .map(([key, value]) => `${formatLogKey(key)}: ${formatLogValue(value)}`)
    .join('；')
}
function formatLogValue(value) {
  if (Array.isArray(value)) return value.map((item) => formatLogValue(item)).join(', ')
  if (typeof value === 'object') {
    const entries = Object.entries(value || {}).filter(([, nestedValue]) => nestedValue !== undefined && nestedValue !== null && nestedValue !== '').slice(0, 3)
    return entries.map(([key, nestedValue]) => `${formatLogKey(key)}=${String(nestedValue)}`).join(', ')
  }
  return String(value)
}

function mapTree(items) {
  return (items || []).map((item) => ({ value: item.id, label: item.name, children: mapTree(item.children || []) }))
}
function flattenTree(items) {
  return (items || []).flatMap((item) => [item, ...flattenTree(item.children || [])])
}
function roleNames(ids = []) {
  return ids.map((id) => {
    const role = roles.value.find((item) => item.id === id)
    return role ? roleDisplayName(role) : ''
  }).filter(Boolean).join('、')
}
function roleDisplayName(role) {
  const name = String(role?.name || '').trim()
  const code = String(role?.code || '').trim().toLowerCase()
  if (name === '销售员' || name === '销售代表' || code === 'r_sales' || code === 'salesperson' || code === 'sales_rep') return t('system.rolePreset.salesperson')
  if (name === backendRoleSeeds.admin.name || code === 'admin' || code === 'tenant_admin') return t('system.rolePreset.admin')
  if (name === backendRoleSeeds.editor.name || code === 'editor') return t('system.rolePreset.editor')
  if (name === backendRoleSeeds.readonly.name || code === 'readonly' || code === 'viewer') return t('system.rolePreset.readonly')
  return name || '-'
}
function isAdminPresetRole(role = {}) {
  const name = String(role?.name || '').trim()
  const code = String(role?.code || '').trim().toLowerCase()
  return name === backendRoleSeeds.admin.name || code === 'admin' || code === 'tenant_admin' || code === 'r_admin'
}
function roleDisplayDescription(role) {
  const description = String(role?.description || '').trim()
  const name = String(role?.name || '').trim()
  const code = String(role?.code || '').trim().toLowerCase()
  if (name === '销售员' || name === '销售代表' || code === 'r_sales' || code === 'salesperson' || code === 'sales_rep') return t('system.rolePreset.salespersonDesc')
  if (description === backendRoleSeeds.admin.description || name === backendRoleSeeds.admin.name || code === 'admin' || code === 'tenant_admin') return t('system.rolePreset.adminDesc')
  if (description === backendRoleSeeds.editor.description || name === backendRoleSeeds.editor.name || code === 'editor') return t('system.rolePreset.editorDesc')
  if (description === backendRoleSeeds.readonly.description || name === backendRoleSeeds.readonly.name || code === 'readonly' || code === 'viewer') return t('system.rolePreset.readonlyDesc')
  return description || '-'
}
function deptName(id) { return flatDepartments.value.find((item) => item.id === id)?.name }
function postName(id) { return posts.value.find((item) => item.id === id)?.name }
function cleanParams(params) {
  return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null))
}
function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value).replace('T', ' ').slice(0, 19)
  const pad = (n) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}
function menuTypeLabel(type) {
  return ({
    catalog: t('system.menuType.catalog'),
    menu: t('system.menuType.menu'),
    button: t('system.menuType.button'),
    iframe: t('system.menuType.iframe'),
    external: t('system.menuType.external')
  }[type] || type)
}
function menuTagType(type) { return ({ catalog: 'info', menu: 'primary', button: 'warning', iframe: 'success', external: 'warning' }[type] || 'info') }
function logActionMap() {
  return {
    ADD_TICKET_NOTE: t('operation.add_ticket_note'),
    SYNC_TICKET_BUG_STATUS: t('operation.sync_ticket_bug_status'),
    SYNC_EXCHANGE_RATES: t('operation.sync_exchange_rates'),
    ADD_CAMPAIGN_FOLLOW_UP: t('operation.add_campaign_follow_up'),
    ADD_LEAD_FOLLOW_UP: t('operation.add_lead_follow_up'),
    ADD_OPPORTUNITY_FOLLOW_UP: t('operation.add_opportunity_follow_up'),
    APPLY_COMPANY_RESEARCH: t('operation.apply_company_research'),
    GENERATE_HOT_INSIGHTS: t('operation.generate_hot_insights'),
    BROADCAST_SYSTEM_NOTIFICATION: t('operation.broadcast_system_notification'),
    CLEANUP_OPERATION_LOGS: t('operation.cleanup_operation_logs'),
    SAVE_COMPANY_SALES_TARGET: t('operation.save_company_sales_target'),
    ALLOCATE_SALES_TARGETS: t('operation.allocate_sales_targets'),
    SAVE_MY_SALES_PLAN: t('operation.save_my_sales_plan'),
    SAVE_ORDER_PAYMENT_PLAN: t('operation.save_order_payment_plan'),
    INVOICE_ORDER: t('operation.invoice_order'),
    CLOSE_TICKET: t('operation.close_ticket'),
    CHANGE_TICKET_STATUS: t('operation.change_ticket_status'),
    ASSIGN_TICKET: t('operation.assign_ticket'),
    TRANSFER_TICKET: t('operation.transfer_ticket'),
    AUTO_ASSIGN_TICKET: t('operation.auto_assign_ticket'),
    CREATE_TICKET_FROM_ORDER: t('operation.create_ticket_from_order'),
    MARK_ORDER_PAID: t('operation.mark_order_paid'),
    SHIP_ORDER: t('operation.ship_order'),
    RETURN_ORDER: t('operation.return_order'),
    CHANGE_ORDER_STATUS: t('operation.change_order_status'),
    FULFILL_ORDER: t('operation.fulfill_order'),
    CREATE_ORDER_FROM_DEAL: t('operation.create_order_from_deal'),
    CREATE_ORDER_FROM_QUOTE: t('operation.create_order_from_quote'),
    CREATE_CRM_OBJECT: t('operation.create_crm_object'),
    UPDATE_CRM_OBJECT: t('operation.update_crm_object'),
    DELETE_CRM_OBJECT: t('operation.delete_crm_object'),
    CREATE_CRM_RELATION: t('operation.create_crm_relation'),
    DELETE_CRM_RELATION: t('operation.delete_crm_relation'),
    SYNC_AI_PRICING: t('operation.sync_ai_pricing'),
    SYNC_HR: t('operation.sync_hr'),
    LIST_PLATFORM_TENANT_SUMMARY: t('operation.list_platform_tenant_summary'),
    LIST_PLATFORM_SETTINGS: t('operation.list_platform_settings'),
    SAVE_PLATFORM_SETTING_DRAFT: t('operation.save_platform_setting_draft'),
    PUBLISH_PLATFORM_SETTING: t('operation.publish_platform_setting'),
    ROLLBACK_PLATFORM_SETTING: t('operation.rollback_platform_setting'),
    SECURITY_RATE_LIMIT: t('operation.security_rate_limit'),
    SECURITY_SCAN_BLOCK: t('operation.security_scan_block'),
    SECURITY_EXPORT_BLOCK: t('operation.security_export_block'),
    SECURITY_SESSION_REVOKED: t('operation.security_session_revoked'),
    EXPORT_OPERATION_LOGS: t('operation.export_operation_logs'),
    EXPORT_LOGIN_LOGS: t('operation.export_login_logs'),
    EXPORT_ORDER_BACKUP: t('operation.export_order_backup')
  }
}
function operationLabel(value = '') {
  const text = String(value || '').trim()
  const upper = text.toUpperCase()
  if (!upper) return '-'
  if (logActionMap()[upper]) return logActionMap()[upper]
  if (upper.includes('LOGIN')) return t('system.operation.loginSuccess')
  if (upper.includes('LOGOUT')) return t('logout')
  if (upper.includes('SYNC')) return t('operation.verb.sync')
  if (upper.includes('ADD')) return t('operation.verb.add')
  if (upper.includes('ASSIGN')) return t('operation.verb.assign')
  if (upper.includes('TRANSFER')) return t('operation.verb.transfer')
  if (upper.includes('APPROVE')) return t('operation.verb.approve')
  if (upper.includes('EXPORT')) return t('common.action.export')
  if (upper.includes('CREATE')) return t('common.action.create')
  if (upper.includes('UPDATE')) return t('common.action.edit')
  if (upper.includes('DELETE')) return t('common.action.delete')
  return formatLogKey(text)
}
function parseLogParams(row) {
  if (!row?.params_json) return {}
  try {
    return JSON.parse(row.params_json)
  } catch {
    return {}
  }
}
function extractTargetInfo(target = '') {
  const match = String(target || '').match(/^(.+?)\[(.+?)\]$/)
  if (!match) return { type: String(target || ''), id: '' }
  return { type: match[1], id: match[2] }
}
function logModuleMap() {
  return {
    order_finance: t('operation.module.order_finance'),
    'order_finance.default': t('operation.module.order_finance'),
    operation_logs: t('operation.module.operation_logs'),
    dashboards: t('nav.销售大屏'),
    sales_target: t('operation.module.sales_target'),
    sales_targets: t('operation.module.sales_target'),
    platform: t('operation.module.platform'),
    'platform.settings': t('operation.module.platform_settings'),
    'platform.tenants.summary': t('operation.module.tenant_summary'),
    security: t('system.security.module'),
    deepseek: t('operation.module.ai_pricing'),
    'deepseek.pricing': t('operation.module.ai_pricing'),
    HR: t('operation.module.hr_sync'),
    hr: t('operation.module.hr_sync'),
    工单: t('domain.auto.17e2f8d878'),
    线索: t('domain.auto.8de0b8fb1d'),
    商机: t('domain.auto.2559c4e3c0'),
    活动: t('domain.auto.5fa95b1b73'),
    租户: t('domain.auto.1c264fa03e'),
    用户: t('domain.auto.baf84751a2'),
    关联: t('operation.module.relation')
  }
}
function moduleFromOperation(operation = '') {
  const upper = String(operation || '').toUpperCase()
  if (upper.includes('SALES_TARGET') || upper.includes('SALES_PLAN')) return t('operation.module.sales_target')
  if (upper.includes('EXCHANGE_RATE') || upper.includes('ORDER_FINANCE')) return t('operation.module.order_finance')
  if (upper.includes('TICKET')) return t('domain.auto.17e2f8d878')
  if (upper.includes('LEAD')) return t('domain.auto.8de0b8fb1d')
  if (upper.includes('OPPORTUNITY')) return t('domain.auto.2559c4e3c0')
  if (upper.includes('CAMPAIGN')) return t('domain.auto.5fa95b1b73')
  if (upper.includes('PRODUCT')) return t('domain.auto.76363b4539')
  if (upper.includes('PROJECT')) return t('domain.auto.6dd16d11ee')
  if (upper.includes('ORDER')) return t('domain.auto.81b3798beb')
  if (upper.includes('TENANT') || upper.includes('PLATFORM')) return t('domain.auto.1c264fa03e')
  if (upper.includes('USER') || upper.includes('PASSWORD') || upper.includes('ACCOUNT')) return t('domain.auto.baf84751a2')
  if (upper.includes('ROLE')) return t('domain.auto.3d0d88d5d4')
  if (upper.includes('LOG')) return t('domain.auto.1db2e8417b')
  return ''
}
function logModuleLabel(row) {
  const operation = String(row?.operation || '').toUpperCase()
  const targetInfo = extractTargetInfo(row?.target)
  const targetType = targetInfo.type
  const targetText = String(row?.target || '').trim()
  if (operation.includes('LOGIN') || operation.includes('LOGOUT') || [t('domain.auto.99e00928fe'), t('domain.auto.43636f9504')].includes(targetType)) return t('domain.auto.d003db6512')
  if (operation.startsWith('SECURITY_')) return t('system.security.module')
  if (operation.includes('ROLE') || targetType === t('domain.auto.6b26695e4d')) return t('domain.auto.3d0d88d5d4')
  if (operation.includes('USER') || operation.includes('PASSWORD') || targetType === t('domain.auto.9ba763ea34')) return t('domain.auto.baf84751a2')
  if (operation.includes('DEPARTMENT') || targetType === t('domain.auto.91061a56c0')) return t('domain.auto.eca7562269')
  if (operation.includes('POST') || targetType === t('domain.auto.f0f3e908c2')) return t('domain.auto.87fc305b3b')
  if (operation.includes('MENU') || targetType === t('domain.auto.7082456207')) return t('domain.auto.d0e839aaff')
  if (operation.includes('TENANT') || targetType === t('domain.auto.cc04fa896e')) return t('domain.auto.1c264fa03e')
  if (logModuleMap()[targetText]) return logModuleMap()[targetText]
  if (logModuleMap()[targetType]) return logModuleMap()[targetType]
  const operationModule = moduleFromOperation(operation)
  if (operationModule) return operationModule
  return businessLabel(targetType) || businessLabel(targetText) || '-'
}
function logTargetLabel(row) {
  const params = parseLogParams(row)
  const targetInfo = extractTargetInfo(row?.target)
  const operation = String(row?.operation || '').toUpperCase()
  const targetText = String(row?.target || '').trim()
  const roleID = Number(params.role_id || targetInfo.id || 0)
  const userID = Number(params.user_id || targetInfo.id || row?.user_id || 0)
  if (operation.includes('ROLE')) {
    const role = roleID ? roleLookup.value.get(roleID) : null
    return params.role_name || role?.name || t('domain.auto.6b26695e4d')
  }
  if (operation.includes('USER') || operation.includes('PASSWORD') || targetInfo.type === t('domain.auto.9ba763ea34')) {
    const user = userID ? userLookup.value.get(userID) : null
    return params.username || user?.name || user?.username || row?.username || t('domain.auto.9ba763ea34')
  }
  if (operation.includes('TENANT')) {
    return params.tenant_name || t('domain.auto.cc04fa896e')
  }
  if (operation.includes('EXCHANGE_RATE') || targetText === 'order_finance.default') return t('operation.target.default_exchange_rate')
  if (operation.includes('SALES_TARGET') || operation.includes('SALES_PLAN')) return targetText || t('operation.module.sales_target')
  if (params.order_id) return `${t('domain.auto.81b3798beb')} #${params.order_id}`
  if (params.ticket_id) return `${t('domain.auto.17e2f8d878')} #${params.ticket_id}`
  if (params.lead_id) return `${t('domain.auto.8de0b8fb1d')} #${params.lead_id}`
  if (params.opportunity_id) return `${t('domain.auto.2559c4e3c0')} #${params.opportunity_id}`
  if (params.campaign_id) return `${t('domain.auto.5fa95b1b73')} #${params.campaign_id}`
  if (params.name) return params.name
  if (params.title) return params.title
  if (params.company_name) return params.company_name
  if (params.domain) return `${businessLabel(params.domain)}${t('domain.auto.9058a1c2c5')}`
  if (targetInfo.type && targetInfo.id) return `${businessLabel(targetInfo.type)} #${targetInfo.id}`
  return businessLabel(targetText) || ''
}
function logDescription(row) {
  if (row?.reason) return row.reason
  const target = logTargetLabel(row)
  return target ? `${operationLabel(row?.operation)}：${target}` : operationLabel(row?.operation)
}
function businessLabel(value = '') {
  const text = String(value || '').trim()
  if (!text) return ''
  const withoutID = extractTargetInfo(text).type || text
  const firstPart = withoutID.split('.')[0]
  return logModuleMap()[text] || logModuleMap()[withoutID] || permissionLabel(text) || permissionLabel(withoutID) || permissionLabel(firstPart) || formatLogKey(text)
}
function formatLogKey(value = '') {
  const text = String(value || '').trim()
  if (!text) return ''
  const knownWords = {
    ADD: t('operation.word.add'),
    SYNC: t('operation.word.sync'),
    SAVE: t('operation.word.save'),
    ALLOCATE: t('operation.word.allocate'),
    COMPANY: t('operation.word.company'),
    SALES: t('operation.word.sales'),
    TARGET: t('operation.word.target'),
    TARGETS: t('operation.word.target'),
    PLAN: t('operation.word.plan'),
    PAYMENT: t('operation.word.payment'),
    INVOICE: t('operation.word.invoice'),
    CLOSE: t('operation.word.close'),
    CHANGE: t('operation.word.change'),
    ASSIGN: t('operation.word.assign'),
    TRANSFER: t('operation.word.transfer'),
    AUTO: t('operation.word.auto'),
    MARK: t('operation.word.mark'),
    PAID: t('operation.word.paid'),
    SHIP: t('operation.word.ship'),
    FULFILL: t('operation.word.fulfill'),
    RETURN: t('operation.word.return'),
    EXCHANGE: t('operation.word.exchange'),
    RATES: t('operation.word.rates'),
    TICKET: t('operation.word.ticket'),
    NOTE: t('operation.word.note'),
    ORDER: t('operation.word.order'),
    FINANCE: t('operation.word.finance'),
    DEFAULT: t('operation.word.default'),
    CRM: t('operation.word.crm'),
    OBJECT: t('operation.word.object'),
    RELATION: t('operation.word.relation'),
    CREATE: t('operation.word.create'),
    UPDATE: t('operation.word.update'),
    DELETE: t('operation.word.delete'),
    CLEANUP: t('operation.word.cleanup'),
    LOGS: t('operation.word.logs')
  }
  const converted = text
    .replace(/[.-]/g, '_')
    .split('_')
    .map((part) => knownWords[part.toUpperCase()] || part)
    .filter(Boolean)
    .join('')
  return converted || text
}
function browserFromUA(ua = '') {
  const lower = ua.toLowerCase()
  if (lower.includes('edg/')) return 'Edge'
  if (lower.includes('firefox')) return 'Firefox'
  if (lower.includes('chrome')) return 'Chrome'
  if (lower.includes('safari')) return 'Safari'
  return t('system.unknown')
}
function osFromUA(ua = '') {
  const lower = ua.toLowerCase()
  if (lower.includes('windows')) return 'Windows'
  if (lower.includes('mac')) return 'macOS'
  if (lower.includes('linux')) return 'Linux'
  return t('system.unknown')
}
function escapeHtml(text) {
  return text.replace(/[&<>"']/g, (char) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]))
}
</script>

<style scoped>
:global(.system-log-detail-message) {
  width: min(640px, calc(100vw - 32px));
}

:global(.system-log-detail) {
  display: grid;
  gap: 10px;
  max-height: 56vh;
  overflow: auto;
}

:global(.system-log-detail__row) {
  display: grid;
  grid-template-columns: 116px minmax(0, 1fr);
  gap: 12px;
  align-items: start;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-extra-light);
}

:global(.system-log-detail__row span) {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

:global(.system-log-detail__row strong) {
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 600;
  line-height: 1.5;
  word-break: break-word;
}
</style>
