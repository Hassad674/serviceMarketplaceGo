"use client"

import { useTranslations } from "next-intl"
import { Modal } from "@/shared/components/ui/modal"
import { Button } from "@/shared/components/ui/button"

type LogoutConfirmDialogProps = {
	open: boolean
	onClose: () => void
	onConfirm: () => void | Promise<void>
}

export function LogoutConfirmDialog({
	open,
	onClose,
	onConfirm,
}: LogoutConfirmDialogProps) {
	const t = useTranslations("common")

	return (
		<Modal open={open} onClose={onClose} title={t("signOutConfirmTitle")}>
			<p className="text-sm leading-relaxed text-muted-foreground">
				{t("signOutConfirmBody")}
			</p>
			<div className="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
				<Button
					type="button"
					variant="outline"
					size="md"
					onClick={onClose}
				>
					{t("cancel")}
				</Button>
				<Button
					type="button"
					variant="primary"
					size="md"
					onClick={onConfirm}
				>
					{t("signOutConfirm")}
				</Button>
			</div>
		</Modal>
	)
}
