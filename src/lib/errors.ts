import { APIError } from '../api/client';

const messages: Record<string, string> = {
  insufficient_capacity:'当前可用存储空间或在线存储节点不足，无法上传此文件。请稍后重试或联系管理员检查存储容量。',
  last_admin:'必须保留至少一个可用管理员。',
  grant_not_permanent:'来源授权不是永久授权，请选择有限时长或请文件所有者重新授权。',
  mount_conflict:'用户网盘的目标位置已占用，请选择其他目录路径。',
  owner_unavailable:'接收用户不可用，请刷新后重新选择。',
  version_conflict:'配置或资源已变化，请刷新后重试。',
  restore_expired:'已超过 30 天恢复期限，不能恢复该用户。',
  account_purged:'该用户的数据已清理，不能恢复。',
  status_not_configured:'服务详情暂不可用，请联系运维人员。',

  path_exists: '同名文件或文件夹已存在，请换一个名称。',
  not_found: '文件或文件夹已不存在，请刷新列表。',
  parent_unavailable: '上级文件夹不可用，请先恢复或选择其他位置。',
  path_unavailable: '此位置暂不可用，请恢复文件或选择其他位置。',
  target_unavailable: '目标文件夹不存在，请重新选择。',
  source_trashed: '请先从回收站恢复此项目。',
  subtree_trashed: '请先恢复此文件夹中已删除的项目。',
  invalid_target: '请选择其他目标位置。',
  invalid_path: '名称或路径不符合要求，请修改后重试。',
  invalid_selection: '请选择 1 至 100 个项目。',
  trash_expired: '已超过恢复期限，无法恢复此文件。',
  trash_changed: '所选项目已发生变化，请关闭窗口并刷新回收站。',
  upload_expired: '此次上传已失效，请确认保存位置后重试。',
  idempotency_conflict: '操作内容已变化，请关闭窗口后重新操作。',
};
const localMessages: Record<string, string> = {
  '目录项目较多，请搜索后再排序。': '目录项目较多，请搜索后再排序。',
  '浏览器可用下载空间不足，请释放本机空间或使用 CLI': '浏览器可用下载空间不足，请释放本机空间或使用命令行客户端下载。',
  '浏览器下载空间不足': '浏览器可用下载空间不足，请清理暂存或使用命令行客户端下载。',
  '密码不正确': '密码不正确，请重试。',
  '名称不能为空或包含斜杠': '请输入有效名称，名称中不能包含斜杠。',
  '文件超过 10000 个块': '文件过大，单个文件最多支持约 39 GB。',
  '当前浏览器下载上限为 256 MiB；请使用支持文件写入的浏览器或 CLI': '当前浏览器无法保存这么大的文件，请更换浏览器或使用命令行客户端下载。',
  '服务端协议与网页版本不兼容': '页面版本已更新，请刷新后重新登录。',
};

export function displayError(error: unknown, fallback = '操作未完成，请稍后重试。'): string {
  if (error instanceof APIError) {
    if (messages[error.code]) return messages[error.code];
    if (error.status === 401) return '登录已失效，请重新登录。';
    if (error.status === 403) return '暂时无法完成此操作，请刷新页面后重试。';
    if (error.status === 409) return '文件状态已变化，请刷新后重试。';
    if (error.status === 429) return '操作较频繁，请稍后重试。';
    if (error.status >= 500) return '服务暂时不可用，请稍后重试。';
  }
  if (error instanceof Error && localMessages[error.message]) return localMessages[error.message];
  if (error instanceof DOMException && error.name === 'NotAllowedError') return '浏览器未允许保存文件，请检查站点下载权限后重试。';
  if (error instanceof DOMException && error.name === 'QuotaExceededError') return '浏览器可用空间不足，请清理下载暂存或释放本机空间后重试。';
  if (error instanceof DOMException && error.name === 'TimeoutError') return '等待时间较长，请重试。';
  if (error instanceof TypeError && /fetch/i.test(error.message)) return '连接失败，请检查网络后重试。';
  return fallback;
}
