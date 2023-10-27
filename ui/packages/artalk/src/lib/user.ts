import type { LocalUser, ContextApi } from '~/types'

class User {
  public ctx?: ContextApi
  public data: LocalUser

  public constructor() {
    // 从 localStorage 导入
    const localUser = JSON.parse(window.localStorage.getItem('ArtalkUser') || '{}')
    this.data = {
      nick: localUser.nick || '',
      email: localUser.email || '',
      link: localUser.link || '',
      token: localUser.token || '',
      isAdmin: localUser.isAdmin || false
    }
  }

  public setContext(ctx: ContextApi) {
    this.ctx = ctx
  }

  /** 保存用户到 localStorage 中 */
  public update(obj: Partial<LocalUser> = {}) {
    Object.entries(obj).forEach(([key, value]) => {
      this.data[key] = value
    })

    window.localStorage.setItem('ArtalkUser', JSON.stringify(this.data))
    this.ctx?.trigger('user-changed', this.data)
  }

  /**
   * Logout
   *
   * @description Logout will clear login status, but not clear user data (nick, email, link)
   */
  public logout() {
    this.update({
      token: '',
      isAdmin: false
    })
  }

  /** 是否已填写基本用户信息 */
  public checkHasBasicUserInfo() {
    return !!this.data.nick && !!this.data.email
  }
}

const UserInstance = new User()

export default UserInstance
