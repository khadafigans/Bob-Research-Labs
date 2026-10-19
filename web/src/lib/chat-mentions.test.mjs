import { activeMention, mentionSearch, mentionToken, selectedMentions } from "./chat-mentions.ts";
import assert from "node:assert/strict";
import test from "node:test";

test("mention trigger supports Chinese and cursor placement without hijacking email", () => {
  assert.equal(activeMention("user@example.com", 16), null);
  assert.equal(activeMention(" @[#1 X]", 12), null);
  assert.deepEqual(activeMention("@ ", 5), { start: 2, end: 5, query: "" });
  assert.equal(activeMention("@\n", 8), null);
});

test("categories, Chinese aliases, IP and keyword search", () => {
  assert.equal(mentionSearch("").categories.length, 9);
  assert.equal(mentionSearch("").categories[0].kind, "finding");
  assert.equal(mentionSearch("").kind, "finding");
  assert.equal(mentionSearch("SQL").query, "SQL");
  assert.equal(mentionSearch("ip 192.0.2.1").kind, "ip");
  assert.equal(mentionSearch(" GET /api").query, "GET /api");
  assert.equal(mentionSearch("acme.com").kind, "");
});

test("tokens roundtrip labels and removing one reference preserves its neighbors", () => {
  const first = mentionToken({ kind: "finding", id: 12, label: "[1]\n" });
  const second = mentionToken({ kind: "ip", id: 13, label: "192.0.2.1" });
  const value = ` ${first}  ${second}`;
  const selected = selectedMentions(value);
  assert.equal(selected.length, 2);
  assert.equal(selected[0].kind, "finding");
  assert.equal(selected[0].id, "12");
  assert.equal(selected[0].text, "（1） ");
  assert.equal(selected[1].kind, "ip");
  const next = value.slice(0, selected[0].start) + value.slice(selected[0].start + selected[0].token.length);
  assert.equal(selectedMentions(next)[0].token, second);
});
