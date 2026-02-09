import './style.css'
import { setupEventListeners } from './ui';
import { apiRequest, ENDPOINTS, setServerUrl } from './api';

const params = new URLSearchParams(window.location.search);

setServerUrl(params.get('server') ?? "");

document.getElementById("head-name")!.innerText = params.get("app") ?? "Auth Service";

async function init() {
  let checkResult = await apiRequest(ENDPOINTS.INIT, {});

  if (!checkResult.success) {
    document.getElementById("form-body")!.innerHTML = `
      <h1 class="body-err">无法连接到服务器</h1>
    `
    console.error("无法连接到服务器");
    return;
  }

  // 设置事件监听
  setupEventListeners();
  console.debug('页面初始化完成', 'success');
}

document.addEventListener('DOMContentLoaded', init);