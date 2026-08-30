import { describe, it, expect, vi, afterEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/svelte"
import ForgotPasswordForm from "./ForgotPasswordForm.svelte"

vi.mock("$lib/api/registration.js", async () => {
  const actual = await vi.importActual("../api/registration.js")
  return { ...actual, requestPasswordReset: vi.fn() }
})

import { requestPasswordReset } from "$lib/api/registration.js"

describe("ForgotPasswordForm", () => {
  afterEach(() => vi.clearAllMocks())

  it("rejects an empty login name", async () => {
    render(ForgotPasswordForm)

    await fireEvent.click(screen.getByRole("button", { name: /send reset link/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent(/enter your login name/i)
    expect(requestPasswordReset).not.toHaveBeenCalled()
  })

  it("submits and shows the check-your-email message", async () => {
    requestPasswordReset.mockResolvedValueOnce({})
    render(ForgotPasswordForm)

    await fireEvent.input(screen.getByPlaceholderText("Login name"), { target: { value: "minnie" } })
    await fireEvent.click(screen.getByRole("button", { name: /send reset link/i }))

    expect(await screen.findByText(/reset link is on its way/i)).toBeInTheDocument()
    expect(requestPasswordReset).toHaveBeenCalledWith("minnie")
  })
})
