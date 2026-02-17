import { greet } from "./index";

describe("greet", () => {
  it("returns a greeting", () => {
    expect(greet("world")).toBe("Hello, world!");
  });
});
