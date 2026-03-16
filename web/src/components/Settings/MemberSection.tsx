import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { PlusIcon } from "lucide-react";
import { useState } from "react";
import toast from "react-hot-toast";
import ConfirmDialog from "@/components/ConfirmDialog";
import { Button } from "@/components/ui/button";
import { userServiceClient } from "@/connect";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useDialog } from "@/hooks/useDialog";
import { useDeleteUser } from "@/hooks/useUserQueries";
import { State } from "@/types/proto/api/v1/common_pb";
import type { User } from "@/types/proto/api/v1/user_service_pb";
import { useTranslate } from "@/utils/i18n";
import CreateUserDialog from "../CreateUserDialog";
import PaginatedMemberTable from "./PaginatedMemberTable";
import SettingSection from "./SettingSection";

const MemberSection = () => {
  const t = useTranslate();
  const currentUser = useCurrentUser();
  const deleteUserMutation = useDeleteUser();
  const createDialog = useDialog();
  const editDialog = useDialog();
  const [editingUser, setEditingUser] = useState<User | undefined>();
  const [archiveTarget, setArchiveTarget] = useState<User | undefined>(undefined);
  const [deleteTarget, setDeleteTarget] = useState<User | undefined>(undefined);

  const handleCreateUser = () => {
    setEditingUser(undefined);
    createDialog.open();
  };

  const handleEditUser = (user: User) => {
    setEditingUser(user);
    editDialog.open();
  };

  const handleArchiveUserClick = async (user: User) => {
    setArchiveTarget(user);
  };

  const confirmArchiveUser = async () => {
    if (!archiveTarget) return;
    const username = archiveTarget.username;
    await userServiceClient.updateUser({
      user: {
        name: archiveTarget.name,
        state: State.ARCHIVED,
      },
      updateMask: create(FieldMaskSchema, { paths: ["state"] }),
    });
    setArchiveTarget(undefined);
    toast.success(t("setting.member-section.archive-success", { username }));
  };

  const handleRestoreUserClick = async (user: User) => {
    const { username } = user;
    await userServiceClient.updateUser({
      user: {
        name: user.name,
        state: State.NORMAL,
      },
      updateMask: create(FieldMaskSchema, { paths: ["state"] }),
    });
    toast.success(t("setting.member-section.restore-success", { username }));
  };

  const handleDeleteUserClick = async (user: User) => {
    setDeleteTarget(user);
  };

  const confirmDeleteUser = async () => {
    if (!deleteTarget) return;
    const { username, name } = deleteTarget;
    deleteUserMutation.mutate(name);
    setDeleteTarget(undefined);
    toast.success(t("setting.member-section.delete-success", { username }));
  };

  return (
    <SettingSection
      title={t("setting.member-list")}
      actions={
        <Button onClick={handleCreateUser}>
          <PlusIcon className="w-4 h-4 mr-2" />
          {t("common.create")}
        </Button>
      }
    >
      {/* Paginated Member Table */}
      <PaginatedMemberTable
        onEdit={handleEditUser}
        onArchive={handleArchiveUserClick}
        onRestore={handleRestoreUserClick}
        onDelete={handleDeleteUserClick}
        currentUserName={currentUser?.name}
      />

      {/* Create User Dialog */}
      <CreateUserDialog open={createDialog.isOpen} onOpenChange={createDialog.setOpen} onSuccess={() => {}} />

      {/* Edit User Dialog */}
      <CreateUserDialog
        open={editDialog.isOpen}
        onOpenChange={editDialog.setOpen}
        user={editingUser}
        onSuccess={() => {}}
      />

      <ConfirmDialog
        open={!!archiveTarget}
        onOpenChange={(open) => !open && setArchiveTarget(undefined)}
        title={archiveTarget ? t("setting.member-section.archive-warning", { username: archiveTarget.username }) : ""}
        description={archiveTarget ? t("setting.member-section.archive-warning-description") : ""}
        confirmLabel={t("common.confirm")}
        cancelLabel={t("common.cancel")}
        onConfirm={confirmArchiveUser}
        confirmVariant="default"
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(undefined)}
        title={deleteTarget ? t("setting.member-section.delete-warning", { username: deleteTarget.username }) : ""}
        description={deleteTarget ? t("setting.member-section.delete-warning-description") : ""}
        confirmLabel={t("common.delete")}
        cancelLabel={t("common.cancel")}
        onConfirm={confirmDeleteUser}
        confirmVariant="destructive"
      />
    </SettingSection>
  );
};

export default MemberSection;
