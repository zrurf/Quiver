import * as opaque from "@serenity-kit/opaque"
import { disableButton, elements, enableButton, hideLoading, hideMessage, showLoading, showMessage } from "./ui"
import { apiRequest, ENDPOINTS, ERROR_MESSAGES } from "./api";
import { Base64Converter, switchTab } from "./utils";
import { send } from "./js/godot_ipc";

let serverPublicKey: any = null;

// 注册流程
export async function handleRegister(username: string, password: string) {
  try {
    showLoading(elements.registerLoading!);
    disableButton(elements.registerBtn as HTMLButtonElement);
    hideMessage(elements.registerMessage!);

    console.debug(`开始注册: ${username}`, 'info');
    
    // 注册步骤 1: 创建注册请求
    const registrationResult = opaque.client.startRegistration({
        password: password
    });

    // 发送注册初始化请求
    const initResponse = await apiRequest(ENDPOINTS.REGISTER_INIT, {
        username: username,
        registration_request: Base64Converter.toStandard(registrationResult.registrationRequest)
    });

    if (!initResponse.success) {
        showMessage(elements.registerMessage!, `注册失败: ${
            (ERROR_MESSAGES.has(initResponse.status)) ? ERROR_MESSAGES.get(initResponse.status) : initResponse.error
        }`, 'error');
        throw new Error(initResponse.error);
    }

    // 存储服务器公钥
    serverPublicKey = initResponse.data.server_public_key;

    // 注册步骤 2: 完成注册
    const registrationRecord = opaque.client.finishRegistration({
        password: password,
        registrationResponse: Base64Converter.toUrlSafe(initResponse.data.registration_response),
        clientRegistrationState: registrationResult.clientRegistrationState,
        identifiers: {
            client: username,
            server: "quiver"
        },
        keyStretching: {
            "argon2id-custom": {
                "iterations": 3,
                "memory": 64 * 1024,
                "parallelism": 4
            }
        }
    });

    // 发送注册完成请求
    const finalizeResponse = await apiRequest(ENDPOINTS.REGISTER_FINALIZE, {
        username: username,
        registration_record: Base64Converter.toStandard(registrationRecord.registrationRecord)
    });

    if (!finalizeResponse.success) {
        showMessage(elements.registerMessage!, `注册失败: ${
            (ERROR_MESSAGES.has(finalizeResponse.status)) ? ERROR_MESSAGES.get(finalizeResponse.status) : finalizeResponse.error
        }`, 'error');
        throw new Error(finalizeResponse.error);
    }

    showMessage(elements.registerMessage!, '注册成功！3秒后自动跳转登录页面', 'success');
    console.debug(`用户 ${username} 注册成功`, 'success');

    // 自动切换到登录页
    setTimeout(() => {
        enableButton(elements.registerBtn as HTMLButtonElement, '注册');
        switchTab('login');
    }, 3000);
    disableButton(elements.registerBtn as HTMLButtonElement);
  } catch (error: any) {
    enableButton(elements.registerBtn as HTMLButtonElement, '注册');
    console.error(`注册错误: ${error.message}`, 'error');
  } finally {
    hideLoading(elements.registerLoading!);
  }
}

// 登录流程
export async function handleLogin(username: string, password: string) {
  try {
    showLoading(elements.loginLoading!);
    disableButton(elements.loginBtn as HTMLButtonElement);
    hideMessage(elements.loginMessage!);

    console.debug(`开始登录: ${username}`, 'info');
    
    // 登录步骤 1: 创建登录请求
    const startLoginResponse = opaque.client.startLogin({
        password: password
    });

    // 发送登录初始化请求
    const initResponse = await apiRequest(ENDPOINTS.LOGIN_INIT, {
        username: username,
        ke1: Base64Converter.toStandard(startLoginResponse.startLoginRequest)
    });

    if (!initResponse.success) {
        showMessage(elements.loginMessage!, `登录失败: ${
            (ERROR_MESSAGES.has(initResponse.status)) ? ERROR_MESSAGES.get(initResponse.status) : initResponse.error
        }`, 'error');
        throw new Error(initResponse.error);
    }

    // 登录步骤 2: 完成登录
    const finishLoginResponse = opaque.client.finishLogin({
        clientLoginState: startLoginResponse.clientLoginState,
        loginResponse: Base64Converter.toUrlSafe(initResponse.data.ke2),
        password: password,
        identifiers: {
            client: username,
            server: "quiver"
        },
        keyStretching: {
            "argon2id-custom": {
                "iterations": 3,
                "memory": 64 * 1024,
                "parallelism": 4
            }
        }
    });

    if (!finishLoginResponse) {
        showMessage(elements.loginMessage!, `登录失败: 账号或密码错误`, 'error');
        throw new Error("KE2 check failed");
    }

    // 发送登录完成请求
    const finalizeResponse = await apiRequest(ENDPOINTS.LOGIN_FINALIZE, {
        ke3: Base64Converter.toStandard(finishLoginResponse?.finishLoginRequest!),
        mac: initResponse.data.mac,
        username: username
    });

    if (!finalizeResponse.success) {
        showMessage(elements.loginMessage!, `登录失败: ${
            (ERROR_MESSAGES.has(finalizeResponse.status)) ? ERROR_MESSAGES.get(finalizeResponse.status) : finalizeResponse.error
        }`, 'error');
        throw new Error(finalizeResponse.error);
    }

    const { uid, access_token, refresh_token, expire_at } = finalizeResponse.data;
    
    showMessage(elements.loginMessage!, `登录成功！UID: ${uid}`, 'success');
    send('login', {
        uid: String(uid),
        access_token: String(access_token),
        refresh_token: String(refresh_token),
        expire_at: expire_at
    }); // 发送登录信息给 Godot
    console.info(`用户 ${username} 登录成功`, 'success');
    disableButton(elements.loginBtn as HTMLButtonElement);
  } catch (error: any) {
    console.error(`登录错误: ${error.message}`, 'error');
    enableButton(elements.loginBtn as HTMLButtonElement, '登录');
  } finally {
    hideLoading(elements.loginLoading!);
  }
}
