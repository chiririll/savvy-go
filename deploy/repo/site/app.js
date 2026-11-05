document.querySelectorAll(".tabs").forEach(function (box) {
  var tabs = box.querySelectorAll('[role="tab"]');
  var panels = box.querySelectorAll('[role="tabpanel"]');
  var key = "tab-" + box.dataset.remember;
  function show(name) {
    tabs.forEach(function (t) { t.setAttribute("aria-selected", t.dataset.tab === name); });
    panels.forEach(function (p) { p.hidden = p.dataset.tab !== name; });
  }
  tabs.forEach(function (t) {
    t.addEventListener("click", function () {
      show(t.dataset.tab);
      try { localStorage.setItem(key, t.dataset.tab); } catch (e) { }
    });
  });
  var saved = null;
  try { saved = localStorage.getItem(key); } catch (e) { }
  show(box.querySelector('[data-tab="' + saved + '"]') ? saved : tabs[0].dataset.tab);
});

document.querySelectorAll(".code").forEach(function (box) {
  var btn = document.createElement("button");
  btn.className = "copy";
  btn.textContent = "Copy";
  btn.addEventListener("click", function () {
    navigator.clipboard.writeText(box.querySelector("pre").textContent).then(function () {
      btn.textContent = "Copied";
      setTimeout(function () { btn.textContent = "Copy"; }, 1500);
    });
  });
  box.appendChild(btn);
});
