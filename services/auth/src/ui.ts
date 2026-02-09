import { handleLogin, handleRegister } from "./auth";
import { switchTab } from "./utils";

export const elements = {
    tabs: document.querySelectorAll('.tab') as NodeListOf<HTMLElement>,
    tabContents: document.querySelectorAll('.tab-content') as NodeListOf<HTMLElement>,
    loginForm: document.getElementById('login-form'),
    registerForm: document.getElementById('register-form'),
    loginBtn: document.getElementById('login-btn'),
    registerBtn: document.getElementById('register-btn'),
    loginMessage: document.getElementById('login-message'),
    registerMessage: document.getElementById('register-message'),
    loginLoading: document.getElementById('login-loading'),
    registerLoading: document.getElementById('register-loading'),
};

export function showMessage(element: HTMLElement, message: string, type : string = 'info') {
element.textContent = message;
element.className = 'message';
switch (type) {
    case 'success':
element.classList.add('success');
break;
    case 'error':
element.classList.add('error');
break;
    case 'info':
element.classList.add('info');
break;
}
element.style.display = 'block';
}

export function hideMessage(element: HTMLElement) {
    element.style.display = 'none';
}

export function showLoading(loadingElement: HTMLElement) {
    loadingElement.style.display = 'block';
}

export function hideLoading(loadingElement: HTMLElement) {
    loadingElement.style.display = 'none';
}

export function disableButton(button: HTMLButtonElement) {
    button.disabled = true;
    button.innerHTML = '处理中...';
}

export function enableButton(button: HTMLButtonElement, text: string) {
    button.disabled = false;
    button.innerHTML = text;
}

// 事件监听
export function setupEventListeners() {
    // 标签页切换
    elements.tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            switchTab(tab.dataset.tab ?? "login");
        });
    });

    // 登录表单提交
    elements.loginForm?.addEventListener('submit', async (e) => {
        e.preventDefault();

        const username = (document.getElementById('login-username') as HTMLInputElement).value.trim();
        const password = (document.getElementById('login-password') as HTMLInputElement).value;

    if (!username || !password) {
        showMessage(elements.loginMessage!, '请填写用户名和密码', 'error');
        return;
    }

    await handleLogin(username, password);
    });

    // 注册表单提交
    elements.registerForm?.addEventListener('submit', async (e) => {
        e.preventDefault();

        const username = (document.getElementById('register-username') as HTMLInputElement).value.trim();
        const password = (document.getElementById('register-password') as HTMLInputElement).value;
        const passwordConfirm = (document.getElementById('register-password-confirm') as HTMLInputElement).value;

        if (!username || !password) {
            showMessage(elements.registerMessage!, '请填写用户名和密码', 'error');
            return;
        }

        if (password !== passwordConfirm) {
            showMessage(elements.registerMessage!, '两次输入的密码不一致', 'error');
            return;
        }

        if (password.length < 6) {
            showMessage(elements.registerMessage!, '密码长度至少6位', 'error');
            return;
        }

        await handleRegister(username, password);
    });
}