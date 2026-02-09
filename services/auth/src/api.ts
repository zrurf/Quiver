let SERVER_URL: string = "";

export const ENDPOINTS = {
  get INIT() { return `${SERVER_URL}/` },
  get REGISTER_INIT() { return `${SERVER_URL}/api/auth/register-init`; },
  get REGISTER_FINALIZE() { return `${SERVER_URL}/api/auth/register-finalize`; },
  get LOGIN_INIT() { return `${SERVER_URL}/api/auth/login-init`; },
  get LOGIN_FINALIZE() { return `${SERVER_URL}/api/auth/login-finalize`; }
};

export const ERROR_MESSAGES: Map<string, string> = new Map([
    ['ERR_SERVER_ERROR', '服务器内部错误'],
    ['ERR_LOGIN_FAILED', '账号或密码错误'],
    ['ERR_USERNAME_CONFLICT', '用户名冲突']
]);

export function setServerUrl(url: string) {
    SERVER_URL = url;
}

export async function apiRequest(url: string, data: any) {
try {
    console.debug(`请求: ${url}`, 'info');
    const response = await fetch(url, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data)
    });

    const result = await response.json();
    console.debug(`响应: ${JSON.stringify(result)}`, 'info');

    if (result.code === 0) {
        return { success: true, data: result.payload };
    } else {
        return { 
            success: false, 
            error: result.message || '请求失败',
            code: result.code,
            status: result.status
        };
    }
} catch (error: any) {
    console.debug(`网络错误: ${error.message}`, 'error');
    return { 
        success: false, 
        error: '网络请求失败: ' + error.message 
    };
}
}