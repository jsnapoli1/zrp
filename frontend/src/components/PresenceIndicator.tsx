import { usePresence } from "../hooks/usePresence";
import { Avatar } from "./ui/avatar";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./ui/tooltip";
import { Badge } from "./ui/badge";
import { Eye, Edit } from "lucide-react";

interface PresenceIndicatorProps {
  resourceType: string;
  resourceId: string | number;
  action?: "viewing" | "editing";
  className?: string;
}

export function PresenceIndicator({
  resourceType,
  resourceId,
  action = "viewing",
  className = "",
}: PresenceIndicatorProps) {
  const { presence, isConnected } = usePresence(resourceType, resourceId, action);

  if (!isConnected || presence.length === 0) {
    return null;
  }

  return (
    <TooltipProvider>
      <div className={`flex items-center gap-2 ${className}`}>
        <div className="flex -space-x-2">
          {presence.slice(0, 3).map((p) => (
            <Tooltip key={p.user_id}>
              <TooltipTrigger asChild>
                <div className="relative">
                  <Avatar className="h-8 w-8 border-2 border-background">
                    <div className="flex h-full w-full items-center justify-center bg-primary text-primary-foreground text-sm font-medium">
                      {p.username.substring(0, 2).toUpperCase()}
                    </div>
                  </Avatar>
                  {p.action === "editing" && (
                    <div className="absolute -bottom-1 -right-1 rounded-full bg-amber-500 p-0.5">
                      <Edit className="h-3 w-3 text-white" />
                    </div>
                  )}
                  {p.action === "viewing" && (
                    <div className="absolute -bottom-1 -right-1 rounded-full bg-blue-500 p-0.5">
                      <Eye className="h-3 w-3 text-white" />
                    </div>
                  )}
                </div>
              </TooltipTrigger>
              <TooltipContent>
                <p className="font-medium">{p.username}</p>
                <p className="text-xs text-muted-foreground">
                  {p.action === "editing" ? "Editing" : "Viewing"}
                </p>
              </TooltipContent>
            </Tooltip>
          ))}
        </div>
        
        {presence.length > 3 && (
          <Badge variant="secondary" className="text-xs">
            +{presence.length - 3} more
          </Badge>
        )}
        
        {presence.length > 0 && (
          <span className="text-xs text-muted-foreground">
            {presence.length === 1 ? "1 other user" : `${presence.length} other users`}
          </span>
        )}
      </div>
    </TooltipProvider>
  );
}

/**
 * Compact version for showing just the count
 */
export function PresenceCount({
  resourceType,
  resourceId,
  className = "",
}: Omit<PresenceIndicatorProps, "action">) {
  const { presence, isConnected } = usePresence(resourceType, resourceId, "viewing");

  if (!isConnected || presence.length === 0) {
    return null;
  }

  const viewing = presence.filter((p) => p.action === "viewing").length;
  const editing = presence.filter((p) => p.action === "editing").length;

  return (
    <div className={`flex items-center gap-2 text-sm ${className}`}>
      {viewing > 0 && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge variant="outline" className="gap-1">
                <Eye className="h-3 w-3" />
                {viewing}
              </Badge>
            </TooltipTrigger>
            <TooltipContent>
              {viewing} {viewing === 1 ? "user" : "users"} viewing
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}
      
      {editing > 0 && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge variant="outline" className="gap-1 border-amber-500 text-amber-700">
                <Edit className="h-3 w-3" />
                {editing}
              </Badge>
            </TooltipTrigger>
            <TooltipContent>
              {editing} {editing === 1 ? "user" : "users"} editing
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}
    </div>
  );
}
