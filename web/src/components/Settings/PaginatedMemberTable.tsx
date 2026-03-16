import { useEffect, useMemo, useRef, useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useInfiniteUsers, useBatchArchiveUsers, useBatchRestoreUsers, useBatchDeleteUsers, userKeys } from "@/hooks/useUserQueries";
import type { User } from "@/types/proto/api/v1/user_service_pb";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { useTranslate } from "@/utils/i18n";
import { Loader2, Search, ChevronRight, Filter, X, Crown, Gift, AlertCircle, ArrowUpDown, ArrowUp, ArrowDown, Archive, Trash2, RotateCcw, Check, Pencil } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import { format } from "date-fns";
import toast from "react-hot-toast";
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { userServiceClient } from "@/connect";

interface PaginatedMemberTableProps {
  onEdit: (user: User) => void;
  onArchive: (user: User) => void;
  onRestore: (user: User) => void;
  onDelete: (user: User) => void;
  currentUserName?: string;
}

const ROW_HEIGHT = 56;
const VISIBLE_ROWS = 15;
const BUFFER_ROWS = 5;

// Default column widths
const DEFAULT_COLUMN_WIDTHS = {
  checkbox: 40,
  userId: 60,
  username: 110,
  role: 80,
  nickname: 110,
  phone: 100,
  state: 70,
  vip: 80,
  vipExpiry: 100,
  created: 90,
  actions: 80,
};

type RoleFilter = "ALL" | "ADMIN" | "USER";
type StateFilter = "ALL" | "NORMAL" | "ARCHIVED";
type VipFilter = "ALL" | "VIP" | "TRIAL" | "NONE";
type SortField = "created_ts" | "updated_ts" | "username";
type SortDirection = "asc" | "desc";

interface SortConfig {
  field: SortField;
  direction: SortDirection;
}

// Editable cell component
interface EditableCellProps {
  width: number;
  value: string;
  onSave: (value: string) => void;
  className?: string;
  placeholder?: string;
  type?: "text" | "password";
}

function EditableCell({ width, value, onSave, className = "", placeholder = "-", type = "text" }: EditableCellProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(value);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setEditValue(value);
  }, [value]);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isEditing]);

  const handleSave = () => {
    if (editValue !== value) {
      onSave(editValue);
    }
    setIsEditing(false);
  };

  const handleCancel = () => {
    setEditValue(value);
    setIsEditing(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSave();
    } else if (e.key === "Escape") {
      handleCancel();
    }
  };

  if (isEditing) {
    return (
      <div
        className="flex items-center gap-1 px-2 py-1"
        style={{ width, minWidth: width, maxWidth: width }}
      >
        <Input
          ref={inputRef}
          type={type}
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onBlur={handleSave}
          onKeyDown={handleKeyDown}
          className="h-7 text-xs px-2"
        />
        <Button
          size="icon"
          variant="ghost"
          className="h-6 w-6 shrink-0"
          onClick={handleSave}
        >
          <Check className="w-3 h-3" />
        </Button>
      </div>
    );
  }

  return (
    <div
      className={`group flex items-center justify-between px-4 py-3 cursor-pointer hover:bg-muted/30 ${className}`}
      style={{ width, minWidth: width, maxWidth: width }}
      onClick={() => setIsEditing(true)}
    >
      <span className="truncate">{value || placeholder}</span>
      <Pencil className="w-3 h-3 opacity-0 group-hover:opacity-50 transition-opacity shrink-0 ml-1" />
    </div>
  );
}

// Resizable header cell component
interface ResizableHeaderProps {
  width: number;
  minWidth?: number;
  maxWidth?: number;
  onResize: (newWidth: number) => void;
  children: React.ReactNode;
  className?: string;
  onClick?: () => void;
}

function ResizableHeader({ width, minWidth = 60, maxWidth = 500, onResize, children, className = "", onClick }: ResizableHeaderProps) {
  const [isResizing, setIsResizing] = useState(false);
  const startXRef = useRef(0);
  const startWidthRef = useRef(width);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsResizing(true);
    startXRef.current = e.clientX;
    startWidthRef.current = width;
  }, [width]);

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      const delta = e.clientX - startXRef.current;
      const newWidth = Math.max(minWidth, Math.min(maxWidth, startWidthRef.current + delta));
      onResize(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizing, minWidth, maxWidth, onResize]);

  return (
    <div
      className={`relative flex items-center px-4 py-3 select-none ${className}`}
      style={{ width, minWidth: width, maxWidth: width }}
      onClick={onClick}
    >
      <div className="flex-1 truncate">{children}</div>
      <div
        className={`absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/50 ${isResizing ? "bg-primary" : ""}`}
        onMouseDown={handleMouseDown}
        style={{ cursor: isResizing ? "col-resize" : "col-resize" }}
      />
    </div>
  );
}

// Data cell component
interface DataCellProps {
  width: number;
  children: React.ReactNode;
  className?: string;
}

function DataCell({ width, children, className = "" }: DataCellProps) {
  return (
    <div
      className={`px-4 py-3 truncate ${className}`}
      style={{ width, minWidth: width, maxWidth: width }}
    >
      {children}
    </div>
  );
}

export default function PaginatedMemberTable({
  onEdit,
  onArchive,
  onRestore,
  onDelete,
  currentUserName,
}: PaginatedMemberTableProps) {
  const t = useTranslate();
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [roleFilter, setRoleFilter] = useState<RoleFilter>("ALL");
  const [stateFilter, setStateFilter] = useState<StateFilter>("ALL");
  const [vipFilter, setVipFilter] = useState<VipFilter>("ALL");
  const [sortConfig, setSortConfig] = useState<SortConfig>({ field: "created_ts", direction: "desc" });
  const [selectedUsers, setSelectedUsers] = useState<Set<string>>(new Set());
  const containerRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);

  // Column widths state
  const [columnWidths, setColumnWidths] = useState(DEFAULT_COLUMN_WIDTHS);

  const batchArchiveUsers = useBatchArchiveUsers();
  const batchRestoreUsers = useBatchRestoreUsers();
  const batchDeleteUsers = useBatchDeleteUsers();

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(searchQuery);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Clear selection when filters change
  useEffect(() => {
    setSelectedUsers(new Set());
  }, [debouncedSearch, roleFilter, stateFilter, vipFilter, sortConfig]);

  const filter = useMemo(() => {
    const filters: string[] = [];
    
    if (debouncedSearch.trim()) {
      filters.push(`search == '${debouncedSearch}'`);
    }
    
    if (roleFilter !== "ALL") {
      filters.push(`role == '${roleFilter}'`);
    }
    
    if (stateFilter !== "ALL") {
      filters.push(`state == '${stateFilter}'`);
    }
    
    return filters.join(" && ");
  }, [debouncedSearch, roleFilter, stateFilter]);

  const orderBy = useMemo(() => {
    const prefix = sortConfig.direction === "desc" ? "-" : "";
    return `${prefix}${sortConfig.field}`;
  }, [sortConfig]);

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } = useInfiniteUsers(
    {
      pageSize: 50,
      filter,
      orderBy,
    },
    { enabled: true }
  );

  const allUsers = useMemo(() => {
    const users = data?.pages.flatMap((page) => page.users) || [];
    
    if (vipFilter === "ALL") return users;
    
    return users.filter((user) => {
      const vipStatus = user.vipStatus;
      if (!vipStatus) return vipFilter === "NONE";
      
      switch (vipFilter) {
        case "VIP":
          return vipStatus.isVip && vipStatus.vipType === 3;
        case "TRIAL":
          return vipStatus.vipType === 2;
        case "NONE":
          return !vipStatus.isVip || vipStatus.vipType === 1;
        default:
          return true;
      }
    });
  }, [data, vipFilter]);

  const selectableUsers = useMemo(() => {
    return allUsers.filter(user => user.name !== currentUserName);
  }, [allUsers, currentUserName]);

  const totalHeight = allUsers.length * ROW_HEIGHT;
  const startIndex = Math.max(0, Math.floor(scrollTop / ROW_HEIGHT) - BUFFER_ROWS);
  const endIndex = Math.min(
    allUsers.length,
    Math.ceil((scrollTop + VISIBLE_ROWS * ROW_HEIGHT) / ROW_HEIGHT) + BUFFER_ROWS
  );
  const visibleUsers = allUsers.slice(startIndex, endIndex);
  const offsetY = startIndex * ROW_HEIGHT;

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const { scrollTop, scrollHeight, clientHeight } = e.currentTarget;
    setScrollTop(scrollTop);

    if (scrollHeight - scrollTop - clientHeight < ROW_HEIGHT * 5 && hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  };

  const handleSort = (field: SortField) => {
    setSortConfig((current) => ({
      field,
      direction: current.field === field && current.direction === "desc" ? "asc" : "desc",
    }));
  };

  const getSortIcon = (field: SortField) => {
    if (sortConfig.field !== field) {
      return <ArrowUpDown className="w-3.5 h-3.5 text-muted-foreground/50" />;
    }
    return sortConfig.direction === "desc" ? (
      <ArrowDown className="w-3.5 h-3.5 text-primary" />
    ) : (
      <ArrowUp className="w-3.5 h-3.5 text-primary" />
    );
  };

  const toggleUserSelection = (userName: string) => {
    setSelectedUsers(prev => {
      const newSet = new Set(prev);
      if (newSet.has(userName)) {
        newSet.delete(userName);
      } else {
        newSet.add(userName);
      }
      return newSet;
    });
  };

  const toggleAllSelection = () => {
    if (selectedUsers.size === selectableUsers.length) {
      setSelectedUsers(new Set());
    } else {
      setSelectedUsers(new Set(selectableUsers.map(u => u.name)));
    }
  };

  const handleBatchArchive = async () => {
    const names = Array.from(selectedUsers);
    try {
      const result = await batchArchiveUsers.mutateAsync(names);
      toast.success(`Archived ${result.archivedCount} users`);
      if (result.failedNames.length > 0) {
        toast.error(`Failed to archive ${result.failedNames.length} users`);
      }
      setSelectedUsers(new Set());
    } catch {
      toast.error("Failed to archive users");
    }
  };

  const handleBatchRestore = async () => {
    const names = Array.from(selectedUsers);
    try {
      const result = await batchRestoreUsers.mutateAsync(names);
      toast.success(`Restored ${result.restoredCount} users`);
      if (result.failedNames.length > 0) {
        toast.error(`Failed to restore ${result.failedNames.length} users`);
      }
      setSelectedUsers(new Set());
    } catch {
      toast.error("Failed to restore users");
    }
  };

  const handleBatchDelete = async () => {
    const names = Array.from(selectedUsers);
    if (!confirm(`Are you sure you want to delete ${names.length} users? This action cannot be undone.`)) {
      return;
    }
    try {
      const result = await batchDeleteUsers.mutateAsync(names);
      toast.success(`Deleted ${result.deletedCount} users`);
      if (result.failedNames.length > 0) {
        toast.error(`Failed to delete ${result.failedNames.length} users`);
      }
      setSelectedUsers(new Set());
    } catch {
      toast.error("Failed to delete users");
    }
  };

  // Inline edit handlers
  const handleUpdateUser = async (user: User, field: string, value: string) => {
    try {
      const updateMask = [field];

      // Build update object based on field
      const updateData: { name: string; username?: string; displayName?: string; phoneNumber?: string } = { name: user.name };
      if (field === "username") {
        updateData.username = value;
      } else if (field === "display_name") {
        updateData.displayName = value;
      } else if (field === "phone_number") {
        updateData.phoneNumber = value;
      }

      await userServiceClient.updateUser({
        user: updateData,
        updateMask: create(FieldMaskSchema, { paths: updateMask })
      });

      // Refresh user data immediately
      await queryClient.refetchQueries({ queryKey: userKeys.all });

      toast.success("Updated successfully");
    } catch (error) {
      toast.error("Failed to update");
      console.error(error);
    }
  };

  const formatRole = (role: User["role"]) => {
    switch (role) {
      case 2:
        return t("setting.member-section.admin");
      case 3:
        return t("setting.member-section.user");
      default:
        return "-";
    }
  };

  const formatState = (state: User["state"]) => {
    if (state === 2) {
      return (
        <Badge variant="secondary" className="text-xs">
          {t("common.archived")}
        </Badge>
      );
    }
    return (
      <Badge variant="outline" className="text-xs text-muted-foreground">
        Normal
      </Badge>
    );
  };

  // Extract user ID from name (format: "users/{id}")
  const getUserId = (user: User): string => {
    if (!user.name) return "-";
    const parts = user.name.split("/");
    return parts[parts.length - 1] || "-";
  };

  const formatVIPStatus = (user: User) => {
    const vipStatus = user.vipStatus;
    
    if (!vipStatus || !vipStatus.isVip) {
      return <span className="text-muted-foreground text-xs">-</span>;
    }

    const expiresDate = vipStatus.expiresDate ? new Date(Number(vipStatus.expiresDate.seconds) * 1000) : null;
    const isExpired = expiresDate && expiresDate < new Date();

    if (vipStatus.vipType === 2) {
      return (
        <div className="flex items-center gap-1 text-blue-600">
          <Gift className="w-3.5 h-3.5" />
          <span className="text-xs">Trial</span>
        </div>
      );
    }

    if (vipStatus.vipType === 3) {
      if (isExpired || vipStatus.state === 2) {
        return (
          <div className="flex items-center gap-1 text-red-500">
            <AlertCircle className="w-3.5 h-3.5" />
            <span className="text-xs">Expired</span>
          </div>
        );
      }
      
      return (
        <div className="flex items-center gap-1 text-amber-500">
          <Crown className="w-3.5 h-3.5" />
          <span className="text-xs">VIP</span>
        </div>
      );
    }

    return <span className="text-muted-foreground text-xs">-</span>;
  };

  const getVIPExpiryDate = (user: User): Date | null => {
    const vipStatus = user.vipStatus;
    if (!vipStatus || !vipStatus.isVip) {
      return null;
    }
    // For trial users, use trial_end_ts from database (not exposed in API yet)
    // For subscription users, use expires_date
    if (vipStatus.expiresDate) {
      return new Date(Number(vipStatus.expiresDate.seconds) * 1000);
    }
    return null;
  };

  const clearFilters = () => {
    setSearchQuery("");
    setRoleFilter("ALL");
    setStateFilter("ALL");
    setVipFilter("ALL");
    setSortConfig({ field: "created_ts", direction: "desc" });
  };

  const hasActiveFilters = searchQuery || roleFilter !== "ALL" || stateFilter !== "ALL" || vipFilter !== "ALL";
  const hasSelection = selectedUsers.size > 0;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Search and Filter Bar */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Search username, email..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="gap-2">
              <Filter className="w-4 h-4" />
              Role: {roleFilter === "ALL" ? "All" : roleFilter}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem onClick={() => setRoleFilter("ALL")}>All Roles</DropdownMenuItem>
            <DropdownMenuItem onClick={() => setRoleFilter("ADMIN")}>Admin</DropdownMenuItem>
            <DropdownMenuItem onClick={() => setRoleFilter("USER")}>User</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="gap-2">
              <Filter className="w-4 h-4" />
              State: {stateFilter === "ALL" ? "All" : stateFilter}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem onClick={() => setStateFilter("ALL")}>All States</DropdownMenuItem>
            <DropdownMenuItem onClick={() => setStateFilter("NORMAL")}>Normal</DropdownMenuItem>
            <DropdownMenuItem onClick={() => setStateFilter("ARCHIVED")}>Archived</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="gap-2">
              <Crown className="w-4 h-4" />
              VIP: {vipFilter === "ALL" ? "All" : vipFilter}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem onClick={() => setVipFilter("ALL")}>All VIP Status</DropdownMenuItem>
            <DropdownMenuItem onClick={() => setVipFilter("VIP")}>VIP (Paid)</DropdownMenuItem>
            <DropdownMenuItem onClick={() => setVipFilter("TRIAL")}>Trial</DropdownMenuItem>
            <DropdownMenuItem onClick={() => setVipFilter("NONE")}>Non-VIP</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {hasActiveFilters && (
          <Button variant="ghost" size="sm" onClick={clearFilters} className="gap-1">
            <X className="w-4 h-4" />
            Clear
          </Button>
        )}

        <div className="ml-auto text-sm text-muted-foreground">
          {allUsers.length > 0 && (
            <span>
              {allUsers.length} members
              {hasNextPage && "+"}
            </span>
          )}
        </div>
      </div>

      {/* Active Filters Display */}
      {hasActiveFilters && (
        <div className="flex flex-wrap gap-2">
          {searchQuery && (
            <Badge variant="secondary" className="gap-1">
              Search: {searchQuery}
              <X className="w-3 h-3 cursor-pointer" onClick={() => setSearchQuery("")} />
            </Badge>
          )}
          {roleFilter !== "ALL" && (
            <Badge variant="secondary" className="gap-1">
              Role: {roleFilter}
              <X className="w-3 h-3 cursor-pointer" onClick={() => setRoleFilter("ALL")} />
            </Badge>
          )}
          {stateFilter !== "ALL" && (
            <Badge variant="secondary" className="gap-1">
              State: {stateFilter}
              <X className="w-3 h-3 cursor-pointer" onClick={() => setStateFilter("ALL")} />
            </Badge>
          )}
          {vipFilter !== "ALL" && (
            <Badge variant="secondary" className="gap-1">
              VIP: {vipFilter}
              <X className="w-3 h-3 cursor-pointer" onClick={() => setVipFilter("ALL")} />
            </Badge>
          )}
        </div>
      )}

      {/* Batch Actions Bar */}
      {hasSelection && (
        <div className="flex items-center gap-3 p-3 bg-muted rounded-lg">
          <span className="text-sm font-medium">{selectedUsers.size} selected</span>
          <div className="flex-1" />
          <Button
            variant="outline"
            size="sm"
            onClick={handleBatchArchive}
            disabled={batchArchiveUsers.isPending}
            className="gap-1"
          >
            <Archive className="w-4 h-4" />
            Archive
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleBatchRestore}
            disabled={batchRestoreUsers.isPending}
            className="gap-1"
          >
            <RotateCcw className="w-4 h-4" />
            Restore
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={handleBatchDelete}
            disabled={batchDeleteUsers.isPending}
            className="gap-1"
          >
            <Trash2 className="w-4 h-4" />
            Delete
          </Button>
        </div>
      )}

      <div
        ref={containerRef}
        className="border rounded-lg overflow-auto"
        style={{ height: VISIBLE_ROWS * ROW_HEIGHT }}
        onScroll={handleScroll}
      >
        <div style={{ height: totalHeight, position: "relative" }}>
          {/* Header with Sort and Resize */}
          <div
            className="sticky top-0 z-10 flex bg-muted border-b font-medium text-sm"
            style={{ height: ROW_HEIGHT }}
          >
            <div
              className="flex items-center justify-center px-2 py-3"
              style={{ width: columnWidths.checkbox }}
            >
              <Checkbox
                checked={selectableUsers.length > 0 && selectedUsers.size === selectableUsers.length}
                onCheckedChange={toggleAllSelection}
              />
            </div>
            <ResizableHeader
              width={columnWidths.userId}
              onResize={(w) => setColumnWidths(prev => ({ ...prev, userId: w }))}
            >
              ID
            </ResizableHeader>
            <ResizableHeader
              width={columnWidths.username}
              onResize={(w) => setColumnWidths(prev => ({ ...prev, username: w }))}
              className="cursor-pointer hover:bg-muted/80"
              onClick={() => handleSort("username")}
            >
              <div className="flex items-center gap-2">
                {t("common.username")}
                {getSortIcon("username")}
              </div>
            </ResizableHeader>
            <ResizableHeader
              width={columnWidths.role}
              onResize={(w) => setColumnWidths(prev => ({ ...prev, role: w }))}
            >
              {t("common.role")}
            </ResizableHeader>
            <ResizableHeader
              width={columnWidths.nickname}
              onResize={(w) => setColumnWidths(prev => ({ ...prev, nickname: w }))}
            >
              {t("common.nickname")}
            </ResizableHeader>
            <ResizableHeader
              width={columnWidths.phone}
              onResize={(w) => setColumnWidths(prev => ({ ...prev, phone: w }))}
            >
              Phone
            </ResizableHeader>
            <ResizableHeader
              width={columnWidths.state}
              onResize={(w) => setColumnWidths(prev => ({ ...prev, state: w }))}
            >
              State
            </ResizableHeader>
            <ResizableHeader
              width={columnWidths.vip}
              onResize={(w) => setColumnWidths(prev => ({ ...prev, vip: w }))}
            >
              VIP Status
            </ResizableHeader>
            <ResizableHeader
              width={columnWidths.vipExpiry}
              onResize={(w) => setColumnWidths(prev => ({ ...prev, vipExpiry: w }))}
            >
              VIP Expiry
            </ResizableHeader>
            <ResizableHeader
              width={columnWidths.created}
              onResize={(w) => setColumnWidths(prev => ({ ...prev, created: w }))}
              className="cursor-pointer hover:bg-muted/80"
              onClick={() => handleSort("created_ts")}
            >
              <div className="flex items-center gap-2">
                Created
                {getSortIcon("created_ts")}
              </div>
            </ResizableHeader>
            <ResizableHeader
              width={columnWidths.actions}
              onResize={(w) => setColumnWidths(prev => ({ ...prev, actions: w }))}
              className="justify-end text-right"
            >
              Actions
            </ResizableHeader>
          </div>

          <div style={{ transform: `translateY(${offsetY}px)` }}>
            {visibleUsers.map((user) => {
              const isCurrentUser = user.name === currentUserName;
              const isSelected = selectedUsers.has(user.name);
              const createdDate = user.createTime ? new Date(Number(user.createTime.seconds) * 1000) : null;

              return (
                <div
                  key={user.name}
                  className="flex items-center border-b hover:bg-muted/50 text-sm"
                  style={{ height: ROW_HEIGHT }}
                >
                  <div
                    className="flex items-center justify-center px-2 py-3"
                    style={{ width: columnWidths.checkbox }}
                  >
                    {!isCurrentUser && (
                      <Checkbox
                        checked={isSelected}
                        onCheckedChange={() => toggleUserSelection(user.name)}
                      />
                    )}
                  </div>
                  {/* Username - editable */}
                  <DataCell width={columnWidths.userId} className="text-muted-foreground text-xs">
                    {getUserId(user)}
                  </DataCell>
                  <EditableCell
                    width={columnWidths.username}
                    value={user.username}
                    onSave={(value) => handleUpdateUser(user, "username", value)}
                    className="text-foreground"
                  />
                  <DataCell width={columnWidths.role}>
                    {formatRole(user.role)}
                  </DataCell>
                  {/* Nickname - editable */}
                  <EditableCell
                    width={columnWidths.nickname}
                    value={user.displayName || ""}
                    onSave={(value) => handleUpdateUser(user, "display_name", value)}
                    className="text-muted-foreground"
                    placeholder="-"
                  />
                  {/* Phone - editable */}
                  <EditableCell
                    width={columnWidths.phone}
                    value={user.phoneNumber || ""}
                    onSave={(value) => handleUpdateUser(user, "phone_number", value)}
                    className="text-muted-foreground text-xs"
                    placeholder="-"
                  />
                  <DataCell width={columnWidths.state}>
                    {formatState(user.state)}
                  </DataCell>
                  <DataCell width={columnWidths.vip}>
                    {formatVIPStatus(user)}
                  </DataCell>
                  <DataCell width={columnWidths.vipExpiry} className="text-muted-foreground text-xs">
                    {(() => {
                      const expiryDate = getVIPExpiryDate(user);
                      if (!expiryDate) return "-";
                      const isExpired = expiryDate < new Date();
                      return (
                        <span className={isExpired ? "text-red-500" : ""}>
                          {format(expiryDate, "yyyy-MM-dd")}
                          {isExpired && <span className="ml-1">(Expired)</span>}
                        </span>
                      );
                    })()}
                  </DataCell>
                  <DataCell width={columnWidths.created} className="text-muted-foreground text-xs">
                    {createdDate ? format(createdDate, "yyyy-MM-dd") : "-"}
                  </DataCell>
                  <DataCell width={columnWidths.actions} className="text-right">
                    {isCurrentUser ? (
                      <span className="text-muted-foreground text-xs">{t("common.yourself")}</span>
                    ) : (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm" className="h-6 px-2">
                            Actions
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => onEdit(user)}>
                            {t("common.update")}
                          </DropdownMenuItem>
                          {user.state === 2 ? (
                            <>
                              <DropdownMenuItem onClick={() => onRestore(user)}>
                                {t("common.restore")}
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => onDelete(user)}
                                className="text-destructive focus:text-destructive"
                              >
                                {t("setting.member-section.delete-member")}
                              </DropdownMenuItem>
                            </>
                          ) : (
                            <DropdownMenuItem onClick={() => onArchive(user)}>
                              {t("setting.member-section.archive-member")}
                            </DropdownMenuItem>
                          )}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    )}
                  </DataCell>
                </div>
              );
            })}
          </div>

          {allUsers.length === 0 && !isLoading && (
            <div className="flex items-center justify-center h-32 text-muted-foreground">
              {debouncedSearch ? "No results found" : "No members found"}
            </div>
          )}
        </div>
      </div>

      {isFetchingNextPage && (
        <div className="flex items-center justify-center py-2">
          <Loader2 className="w-5 h-5 animate-spin text-primary" />
          <span className="ml-2 text-sm text-muted-foreground">Loading more...</span>
        </div>
      )}

      <div className="flex items-center justify-between">
        <div className="text-sm text-muted-foreground">
          {allUsers.length > 0 && (
            <span>
              Showing {allUsers.length} results
              {hasNextPage && " (more available)"}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => fetchNextPage()}
            disabled={!hasNextPage || isFetchingNextPage}
          >
            {isFetchingNextPage ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <>
                Load more
                <ChevronRight className="w-4 h-4 ml-1" />
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}
