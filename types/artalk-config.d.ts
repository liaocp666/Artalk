export default interface ArtalkConfig {
  /** 装载元素 */
  el: string

  /** 评论框占位字符 */
  placeholder?: string

  /** 评论为空时显示字符 */
  noComment?: string

  /** 发送按钮文字 */
  sendBtn?: string

  /** 页面唯一标识（完整 URL） */
  pageKey: string

  /** 页面标题 */
  pageTitle?: string

  /** 服务器地址 */
  server: string

  /** 站点名 */
  site?: string

  /** 表情包 */
  emoticons?: object|any

  /** Gravatar 头像 */
  gravatar?: {
    /** 镜像 */
    mirror?: string

    /** 默认头像（URL or Gravatar Type） */
    default?: string
  }

  /** 查看更多配置 */
  readMore?: {
    /** 每次请求获取数量 */
    pageSize?: number
    /** 滚动到底部自动加载 */
    autoLoad?: boolean
  }

  /** 评论投票按钮 */
  vote?: boolean

  /** 评论投票反对按钮 */
  voteDown?: boolean

  /** 暗黑模式 */
  darkMode?: boolean

  /** 请求超时（单位：秒） */
  reqTimeout?: number

  /** 平铺模式 */
  flatMode?: boolean

  /** 最大嵌套数 */
  maxNesting?: number

  /** 显示 UA 徽标 */
  uaBadge?: boolean

  /** 版本检测 */
  versionCheck?: boolean
}

export interface LocalUser {
  /** 昵称 */
  nick: string

  /** 邮箱 */
  email: string

  /** 链接 */
  link: string

  /** TOKEN */
  token: string

  /** 是否为管理员 */
  isAdmin: boolean
}
