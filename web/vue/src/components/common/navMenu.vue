<template>
  <div v-cloak>
    <el-menu
      :default-active="currentRoute"
      mode="horizontal"
      background-color="#545c64"
      text-color="#fff"
      active-text-color="#ffd04b"
      router
      class="nav-menu">
      <el-menu-item index="/task">任务管理</el-menu-item>
      <el-menu-item index="/host">任务节点</el-menu-item>
      <el-menu-item v-if="isAdmin" index="/user">用户管理</el-menu-item>
      <el-menu-item v-if="isAdmin" index="/system">系统管理</el-menu-item>
      <el-menu-item v-if="isAdmin" index="/admin">后台管理</el-menu-item>
      <el-submenu v-if="isAdmin" index="llm">
        <template slot="title">AI助手</template>
        <el-menu-item index="/llm/chat">AI对话</el-menu-item>
        <el-menu-item index="/llm/settings">AI配置</el-menu-item>
      </el-submenu>
      <el-menu-item v-else index="/llm/chat">AI助手</el-menu-item>
      <el-submenu v-if="$store.getters.user.token" index="userStatus" class="nav-user-menu">
        <template slot="title">{{ $store.getters.user.username }}</template>
        <el-menu-item index="/user/edit-my-password">修改密码</el-menu-item>
        <el-menu-item @click="logout" index="/user/logout">退出</el-menu-item>
      </el-submenu>
    </el-menu>
  </div>
</template>

<script>

export default {
  name: 'app-nav-menu',
  data () {
    return {}
  },
  computed: {
    isAdmin () {
      return this.$store.getters.user.isAdmin
    },
    currentRoute () {
      if (this.$route.path === '/') {
        return '/task'
      }
      const p = this.$route.path
      if (p.startsWith('/llm/')) {
        return p
      }
      const segments = p.split('/')
      return `/${segments[1]}`
    }
  },
  methods: {
    logout () {
      this.$store.commit('logout')
      this.$router.push('/')
    }
  }
}
</script>

<style scoped>
.nav-menu {
  display: flex;
  align-items: stretch;
}
.nav-user-menu {
  margin-left: auto;
}
</style>
