import httpClient from '../utils/httpClient'

export default {
  getSettings (callback) {
    httpClient.get('/llm/settings', {}, callback)
  },

  updateSettings (params, callback) {
    httpClient.post('/llm/settings', params, callback)
  },

  chat (payload, callback) {
    httpClient.postJSON('/llm/chat', payload, callback)
  }
}
