import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/svelte"
import RegisterForm from "./RegisterForm.svelte"

vi.mock("$lib/api/registration.js", async () => {
  const actual = await vi.importActual("../api/registration.js")
  return { ...actual, register: vi.fn() }
})

import { register } from "$lib/api/registration.js"
import { LoginSessionError } from "$lib/api/loginSession.js"

describe("RegisterForm", () => {
  beforeEach(() => {})
  afterEach(() => vi.clearAllMocks())

  async function fillForm() {
    await fireEvent.input(screen.getByPlaceholderText("Login name"), { target: { value: "minnie" } })
    await fireEvent.input(screen.getByPlaceholderText("Email"), { target: { value: "minnie@example.com" } })
    await fireEvent.input(screen.getByPlaceholderText("First name"), { target: { value: "Minnie" } })
    await fireEvent.input(screen.getByPlaceholderText("Last name"), { target: { value: "Mouse" } })
    await fireEvent.input(screen.getByPlaceholderText("Password"), { target: { value: "hunter2" } })
  }

  it("rejects submission with missing fields", async () => {
    render(RegisterForm)

    await fireEvent.click(screen.getByRole("button", { name: /create account/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent(/fill in every field/i)
    expect(register).not.toHaveBeenCalled()
  })

  it("registers and shows the check-your-email message", async () => {
    register.mockResolvedValueOnce({ userId: "user-1" })
    render(RegisterForm)

    await fillForm()
    await fireEvent.click(screen.getByRole("button", { name: /create account/i }))

    expect(await screen.findByText(/check your email/i)).toBeInTheDocument()
    expect(register).toHaveBeenCalledWith({
      loginName: "minnie",
      email: "minnie@example.com",
      givenName: "Minnie",
      familyName: "Mouse",
      password: "hunter2",
    })
  })

  it("shows an error when the login name or email is already taken", async () => {
    register.mockRejectedValueOnce(new LoginSessionError("already exists", 401))
    render(RegisterForm)

    await fillForm()
    await fireEvent.click(screen.getByRole("button", { name: /create account/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent(/already taken/i)
  })
})
