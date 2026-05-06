<template>
  <el-container>
    <el-main>
      <h3>后台管理 - 权限策略（Casbin RBAC）</h3>

      <el-tabs v-model="activeTab">
        <!-- 策略规则管理 -->
        <el-tab-pane label="策略规则" name="policy">
          <el-form :inline="true" style="margin-bottom:16px">
            <el-form-item label="角色/用户">
              <el-input v-model="newPolicy.sub" placeholder="例: admin" style="width:120px"></el-input>
            </el-form-item>
            <el-form-item label="租户域">
              <el-input v-model="newPolicy.dom" placeholder="例: * 或 tenant1" style="width:100px"></el-input>
            </el-form-item>
            <el-form-item label="资源">
              <el-input v-model="newPolicy.obj" placeholder="例: /task" style="width:100px"></el-input>
            </el-form-item>
            <el-form-item label="操作">
              <el-select v-model="newPolicy.act" style="width:100px">
                <el-option value="*" label="全部(*)"></el-option>
                <el-option value="read" label="read"></el-option>
                <el-option value="write" label="write"></el-option>
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="addPolicy">添加策略</el-button>
            </el-form-item>
          </el-form>

          <el-table :data="policies" border style="width:100%">
            <el-table-column label="角色/用户" prop="0"></el-table-column>
            <el-table-column label="租户域" prop="1"></el-table-column>
            <el-table-column label="资源" prop="2"></el-table-column>
            <el-table-column label="操作" prop="3"></el-table-column>
            <el-table-column label="管理" width="100">
              <template slot-scope="scope">
                <el-button type="danger" size="mini" @click="removePolicy(scope.row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 角色分配管理 -->
        <el-tab-pane label="角色分配" name="role">
          <el-form :inline="true" style="margin-bottom:16px">
            <el-form-item label="用户名">
              <el-input v-model="newRole.user" placeholder="用户名" style="width:120px"></el-input>
            </el-form-item>
            <el-form-item label="角色">
              <el-select v-model="newRole.role" style="width:120px">
                <el-option value="admin" label="admin（管理员）"></el-option>
                <el-option value="operator" label="operator（操作员）"></el-option>
                <el-option value="viewer" label="viewer（只读）"></el-option>
              </el-select>
            </el-form-item>
            <el-form-item label="租户域">
              <el-input v-model="newRole.domain" placeholder="例: * 或 tenant1" style="width:100px"></el-input>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="addRole">分配角色</el-button>
            </el-form-item>
          </el-form>

          <el-table :data="groupings" border style="width:100%">
            <el-table-column label="用户" prop="0"></el-table-column>
            <el-table-column label="角色" prop="1"></el-table-column>
            <el-table-column label="租户域" prop="2"></el-table-column>
            <el-table-column label="管理" width="100">
              <template slot-scope="scope">
                <el-button type="danger" size="mini" @click="removeRole(scope.row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-main>
  </el-container>
</template>

<script>
import adminService from '../../api/admin'

export default {
  name: 'admin-policy',
  data () {
    return {
      activeTab: 'policy',
      policies: [],
      groupings: [],
      newPolicy: {sub: '', dom: '*', obj: '/task', act: 'read'},
      newRole: {user: '', role: 'operator', domain: '*'}
    }
  },
  created () {
    this.loadData()
  },
  methods: {
    loadData () {
      adminService.listPolicies((data) => {
        if (!data) {
          this.$message.error('加载权限策略失败')
          return
        }
        this.policies = data.policies || []
        this.groupings = data.groupings || []
      })
    },
    addPolicy () {
      const {sub, dom, obj, act} = this.newPolicy
      if (!sub || !dom || !obj || !act) {
        this.$message.warning('请填写完整策略信息')
        return
      }
      adminService.addPolicy({sub, dom, obj, act}, () => {
        this.$message.success('添加成功')
        this.loadData()
      })
    },
    removePolicy (row) {
      this.$appConfirm(() => {
        adminService.removePolicy({sub: row[0], dom: row[1], obj: row[2], act: row[3]}, () => {
          this.$message.success('删除成功')
          this.loadData()
        })
      })
    },
    addRole () {
      const {user, role, domain} = this.newRole
      if (!user || !role || !domain) {
        this.$message.warning('请填写完整角色信息')
        return
      }
      adminService.addRoleForUser({user, role, domain}, () => {
        this.$message.success('分配成功')
        this.loadData()
      })
    },
    removeRole (row) {
      this.$appConfirm(() => {
        adminService.removeRoleForUser({user: row[0], role: row[1], domain: row[2]}, () => {
          this.$message.success('删除成功')
          this.loadData()
        })
      })
    }
  }
}
</script>
