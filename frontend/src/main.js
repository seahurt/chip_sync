import { createApp } from "vue";
import ElementPlus from "element-plus";
import "element-plus/dist/index.css";
import * as ElementPlusIconsVue from "@element-plus/icons-vue";
import App from "./App.vue";
import "./style.css";

const app = createApp(App);

// 添加 Vue 全局错误处理
app.config.errorHandler = (err, instance, info) => {
  console.error("Vue 错误:", err, info);
  // 不让错误传播导致界面崩溃
};

// 添加全局未捕获错误处理
window.addEventListener("error", (event) => {
  console.error("全局错误:", event.error);
  event.preventDefault(); // 防止错误导致界面崩溃
});

window.addEventListener("unhandledrejection", (event) => {
  console.error("未处理的 Promise 拒绝:", event.reason);
  event.preventDefault(); // 防止错误导致界面崩溃
});

// 注册所有 Element Plus 图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component);
}

app.use(ElementPlus);
app.mount("#app");
