import httpClient from '../utils/httpClient'

export default {
  listPolicies (callback) {
    httpClient.get('/admin/policy', {}, callback)
  },

  addPolicy (params, callback) {
    httpClient.post('/admin/policy/add', params, callback)
  },

  removePolicy (params, callback) {
    httpClient.post('/admin/policy/remove', params, callback)
  },

  addRoleForUser (params, callback) {
    httpClient.post('/admin/role/add', params, callback)
  },

  removeRoleForUser (params, callback) {
    httpClient.post('/admin/role/remove', params, callback)
  }
}
