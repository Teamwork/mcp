// Runs the timelog app's script against a minimal DOM and a fake host, and
// prints what each scenario observed as JSON. Driven by
// TestTimelogCreateAppNeverLogsTwice; usage: node timelog_app_harness.js <app.html>
"use strict";

const fs = require("fs");
const vm = require("vm");

const html = fs.readFileSync(process.argv[2], "utf8");
const script = html.match(/<script>([\s\S]*)<\/script>/)[1];

const modelInput = {
  task_id: 36434673,
  date: "2026-10-23",
  time: "13:00:00",
  hours: 0,
  minutes: 15,
  description: "MCP duplicate test",
  is_utc: true,
};

// "```" sorts first, as on the site that reported the bug.
const toolBodies = {
  "twprojects-list_projects": { projects: [{ id: 2, name: "```" }, { id: 1406292, name: "Real" }] },
  "twprojects-list_tasks": { tasks: [{ id: 99, name: "Other task" }] },
  "twprojects-get_task": { task: { tasklist: { id: 55, type: "tasklists" } } },
  "twprojects-get_tasklist": { tasklist: { project: { id: 1406292, type: "projects" } } },
};

function element(tag) {
  const node = {
    tag, _value: "", disabled: false, checked: false, textContent: "", className: "",
    style: {}, options: [], listeners: {},
    setAttribute() {},
    addEventListener(type, fn) { (this.listeners[type] ||= []).push(fn); },
    appendChild(child) { this.options.push(child); },
    set innerHTML(_) { this.options = []; },
  };
  Object.defineProperty(node, "value", {
    get() { return this._value; },
    set(v) {
      // A select only takes a value one of its options carries.
      this._value = tag !== "select" || this.options.some((o) => o.value === v) ? v : "";
    },
  });
  return node;
}

function newApp({ createReply }) {
  const ids = ["timelog-form", "submit", "status", "project-id", "task-id", "project-hint", "task-hint",
    "date", "time", "hours", "minutes", "description", "billable", "time-label"];
  const els = Object.fromEntries(ids.map((id) => [id, element(
    { "project-id": "select", "task-id": "select", "timelog-form": "form", submit: "button" }[id] || "input")]));
  const actions = element("div");
  const controls = ["submit", "project-id", "task-id", "date", "time", "hours", "minutes", "description", "billable"];
  els["timelog-form"].querySelector = () => actions;
  els["timelog-form"].querySelectorAll = () => controls.map((id) => els[id]);

  const listeners = [];
  const creates = [];
  const deliver = (data) => listeners.forEach((fn) => fn({ data, origin: "https://host.example.com" }));
  const reply = (id, result) => setTimeout(() => deliver({ jsonrpc: "2.0", id, result }), 1);

  const window = {
    addEventListener(type, fn) { if (type === "message") listeners.push(fn); },
    setTimeout: (fn, ms) => setTimeout(fn, ms),
    parent: {
      postMessage(msg) {
        if (msg.method === "ui/initialize") {
          reply(msg.id, {});
          return;
        }
        if (msg.method !== "tools/call") {
          return;
        }
        const { name, arguments: args } = msg.params;
        if (name === "twprojects-create_timelog") {
          creates.push(args);
          const result = createReply(creates.length);
          if (result) {
            reply(msg.id, result);
          }
          return;
        }
        reply(msg.id, { content: [{ type: "text", text: JSON.stringify(toolBodies[name]) }] });
      },
    },
  };

  vm.runInNewContext(script, {
    window,
    document: {
      getElementById: (id) => els[id],
      createElement: () => ({ value: "", textContent: "" }),
      documentElement: { scrollWidth: 1, scrollHeight: 1 },
    },
    // The app's own request timeout is 30s; collapse it so a scenario can hit it.
    setTimeout: (fn, ms) => setTimeout(fn, ms >= 30000 ? 20 : ms),
  });

  return {
    host: (method, params) => deliver({ jsonrpc: "2.0", method, params }),
    submit: () => els["timelog-form"].listeners.submit.forEach((fn) => fn({ preventDefault() {} })),
    observe: () => ({
      creates,
      project: els["project-id"].value,
      task: els["task-id"].value,
      date: els.date.value,
      submitDisabled: els.submit.disabled,
      locked: actions.style.display === "none",
      status: els.status.textContent,
    }),
  };
}

const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const created = () => ({ content: [{ type: "text", text: "Timelog created successfully with ID 19020699" }] });
const failed = () => ({ content: [{ type: "text", text: "failed to create timelog" }], isError: true });

const scenarios = {
  // The ticket: the model's create succeeds, then the form is submitted.
  "host-success": async (app) => {
    app.host("ui/notifications/tool-input", { arguments: modelInput });
    await wait(40);
    app.host("ui/notifications/tool-result", created());
    app.submit();
  },
  "submit-during-host-call": async (app) => {
    app.host("ui/notifications/tool-input", { arguments: modelInput });
    await wait(40);
    app.submit();
  },
  "host-result-without-content": async (app) => {
    app.host("ui/notifications/tool-input", { arguments: modelInput });
    await wait(40);
    app.host("ui/notifications/tool-result", { structuredContent: {} });
    app.submit();
  },
  "host-error-then-retry": async (app) => {
    app.host("ui/notifications/tool-input", { arguments: { ...modelInput, user_id: 12345, tag_ids: [7] } });
    await wait(40);
    app.host("ui/notifications/tool-result", failed());
    app.submit();
    // The host echoes the form's own call back to it.
    app.host("ui/notifications/tool-input", { arguments: { date: "2030-01-01" } });
    app.host("ui/notifications/tool-result", created());
    await wait(20);
    app.submit();
  },
  "host-cancelled": async (app) => {
    app.host("ui/notifications/tool-input", { arguments: modelInput });
    await wait(40);
    app.host("ui/notifications/tool-cancelled", { reason: "user denied" });
    app.submit();
  },
  "input-after-lock": async (app) => {
    app.host("ui/notifications/tool-input", { arguments: modelInput });
    await wait(40);
    app.host("ui/notifications/tool-result", created());
    app.host("ui/notifications/tool-input", { arguments: { ...modelInput, date: "2030-01-01" } });
  },
  "own-create-times-out": async (app) => {
    app.host("ui/notifications/tool-input", { arguments: modelInput });
    await wait(40);
    app.host("ui/notifications/tool-result", failed());
    app.submit();
    await wait(60);
    app.submit();
  },
  "utc-without-time": async (app) => {
    const { time, ...input } = modelInput;
    app.host("ui/notifications/tool-input", { arguments: input });
    await wait(40);
    app.host("ui/notifications/tool-cancelled", {});
    app.submit();
  },
};

(async () => {
  const out = {};
  for (const [name, run] of Object.entries(scenarios)) {
    const app = newApp({ createReply: name === "own-create-times-out" ? () => null : created });
    await run(app);
    await wait(40);
    out[name] = app.observe();
  }
  // The app leaves its 5s ui/initialize timer behind; do not wait it out.
  process.stdout.write(JSON.stringify(out), () => process.exit(0));
})();
