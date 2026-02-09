/**
 * @param {string} action
 * @param {any} data
 */
export function send(action, data) {
    // @ts-ignore
    return ipc.postMessage(JSON.stringify({
        action: action,
        payload: data,
        ts: Date.now()
    }));
}