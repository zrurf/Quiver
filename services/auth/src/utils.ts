import { elements, hideMessage } from "./ui";

// 标签页切换
export function switchTab(tabName: string) {
    // 更新标签
    elements.tabs.forEach(tab => {
    if (tab.dataset.tab === tabName) {
        tab.classList.add('active');
    } else {
        tab.classList.remove('active');
    }
});

    // 更新内容
    elements.tabContents.forEach(content => {
if (content.id === `${tabName}-tab`) {
    content.classList.add('active');
} else {
    content.classList.remove('active');
}
    });

    // 清空消息
    hideMessage(elements.loginMessage!);
    hideMessage(elements.registerMessage!);
}

export class Base64Converter {
  // URL Base64 转 标准 Base64
  static toStandard(urlBase64: string): string {
    let standard = urlBase64.replace(/-/g, '+').replace(/_/g, '/');
    const padding = standard.length % 4;
    if (padding) {
      standard += '='.repeat(4 - padding);
    }
    return standard;
  }

  // 标准 Base64 转 URL Base64
  static toUrlSafe(standardBase64: string): string {
    return standardBase64
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=+$/, '');
  }

  static strToBase64(str: string) {
    const bytes = new TextEncoder().encode(str); // UTF-8 编码
    const binary = String.fromCharCode(...bytes); // 转二进制字符串
    return btoa(binary); // Base64 编码
  }

  static base64ToStr(base64: string) {
    const binary = atob(base64);
    const bytes = Uint8Array.from(binary, c => c.charCodeAt(0));
    return new TextDecoder().decode(bytes); // UTF-8 解码
  }
}