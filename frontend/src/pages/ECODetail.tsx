import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Separator } from "../components/ui/separator";
import { Skeleton } from "../components/ui/skeleton";
import { 
  ArrowLeft, 
  FileText, 
  Calendar, 
  User, 
  Package,
  CheckCircle,
  XCircle,
  Settings
} from "lucide-react";
import { api, type ECO, type ECORevision } from "../lib/api";

interface ECOWithDetails extends ECO {
  affected_parts?: Array<{
    ipn: string;
    description?: string;
    error?: string;
  }>;
}

const statusConfig = {
  draft: { 
    label: 'Draft', 
    variant: 'secondary' as const, 
    color: 'text-gray-600',
    description: 'ECO is being prepared and not yet submitted for review'
  },
  open: { 
    label: 'Open', 
    variant: 'default' as const, 
    color: 'text-blue-600',
    description: 'ECO is submitted and awaiting approval'
  },
  approved: { 
    label: 'Approved', 
    variant: 'default' as const, 
    color: 'text-green-600',
    description: 'ECO has been approved and can be implemented'
  },
  implemented: { 
    label: 'Implemented', 
    variant: 'outline' as const, 
    color: 'text-green-800',
    description: 'ECO changes have been implemented and are complete'
  },
  rejected: { 
    label: 'Rejected', 
    variant: 'destructive' as const, 
    color: 'text-red-600',
    description: 'ECO was rejected and will not be implemented'
  },
};

function ECODetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [eco, setECO] = useState<ECOWithDetails | null>(null);
  const [revisions, setRevisions] = useState<ECORevision[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  useEffect(() => {
    if (id) {
      fetchECODetails();
      fetchRevisions();
    }
  }, [id]);

  const fetchECODetails = async () => {
    if (!id) return;
    
    setLoading(true);
    try {
      const data = await api.getECO(id);
      setECO(data as ECOWithDetails);
    } catch (error) {
      console.error("Failed to fetch ECO details:", error);
    } finally {
      setLoading(false);
    }
  };

  const fetchRevisions = async () => {
    if (!id) return;
    try {
      const data = await api.getECORevisions(id);
      setRevisions(data);
    } catch (error) {
      console.error("Failed to fetch revisions:", error);
    }
  };

  const handleStatusAction = async (action: 'approve' | 'implement' | 'reject') => {
    if (!id || !eco) return;
    
    setActionLoading(action);
    try {
      switch (action) {
        case 'approve':
          await api.approveECO(id);
          break;
        case 'implement':
          await api.implementECO(id);
          break;
        case 'reject':
          await api.rejectECO(id);
          break;
      }
      
      // Refresh the ECO details to get updated data
      await fetchECODetails();
    } catch (error) {
      console.error(`Failed to ${action} ECO:`, error);
    } finally {
      setActionLoading(null);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getStatusConfig = (status: string) => {
    return statusConfig[status as keyof typeof statusConfig] || statusConfig.draft;
  };

  const canApprove = eco && eco.status === 'open';
  const canImplement = eco && eco.status === 'approved';
  const canReject = eco && ['draft', 'open'].includes(eco.status);

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Skeleton className="h-10 w-10" />
          <Skeleton className="h-8 w-64" />
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            <Skeleton className="h-96" />
            <Skeleton className="h-48" />
          </div>
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (!eco) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" onClick={() => navigate('/ecos')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to ECOs
          </Button>
        </div>
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <FileText className="h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">ECO Not Found</h3>
            <p className="text-muted-foreground text-center">
              The ECO with ID "{id}" could not be found.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const statusConfig_ = getStatusConfig(eco.status);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" onClick={() => navigate('/ecos')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to ECOs
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight font-mono">{eco.id}</h1>
            <p className="text-muted-foreground">
              {eco.title}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Badge variant={statusConfig_.variant} className={statusConfig_.color}>
            {statusConfig_.label}
          </Badge>
          {eco.priority && (
            <Badge variant="outline" className="capitalize">
              {eco.priority} Priority
            </Badge>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* ECO Details */}
          <Card>
            <CardHeader>
              <CardTitle>ECO Details</CardTitle>
              <p className="text-sm text-muted-foreground">
                {statusConfig_.description}
              </p>
            </CardHeader>
            <CardContent className="space-y-6">
              <div>
                <label className="text-sm font-medium text-muted-foreground">Title</label>
                <p className="mt-1 text-lg font-medium">{eco.title}</p>
              </div>

              <Separator />

              <div>
                <label className="text-sm font-medium text-muted-foreground">Description</label>
                <p className="mt-2 whitespace-pre-wrap">{eco.description}</p>
              </div>

              <Separator />

              <div>
                <label className="text-sm font-medium text-muted-foreground">Reason for Change</label>
                <p className="mt-2 whitespace-pre-wrap">{eco.reason}</p>
              </div>
            </CardContent>
          </Card>

          {/* Affected Parts */}
          {eco.affected_parts && eco.affected_parts.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center">
                  <Package className="h-5 w-5 mr-2" />
                  Affected Parts ({eco.affected_parts.length})
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {eco.affected_parts.map((part, index) => (
                    <div 
                      key={index}
                      className="flex items-center justify-between p-3 border rounded-md hover:bg-muted/50 cursor-pointer"
                      onClick={() => !part.error && navigate(`/parts/${encodeURIComponent(part.ipn)}`)}
                    >
                      <div className="flex items-center space-x-3">
                        <Package className="h-4 w-4 text-muted-foreground" />
                        <div>
                          <p className="font-mono font-medium">{part.ipn}</p>
                          {part.description && (
                            <p className="text-sm text-muted-foreground">{part.description}</p>
                          )}
                          {part.error && (
                            <p className="text-sm text-red-600">{part.error}</p>
                          )}
                        </div>
                      </div>
                      {part.error ? (
                        <Badge variant="destructive">Not Found</Badge>
                      ) : (
                        <Badge variant="outline">View Part</Badge>
                      )}
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
          {/* Revision History */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <FileText className="h-5 w-5 mr-2" />
                Revision History
              </CardTitle>
            </CardHeader>
            <CardContent>
              {revisions.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-4">No revisions recorded yet</p>
              ) : (
                <div className="relative">
                  <div className="absolute left-4 top-0 bottom-0 w-px bg-border" />
                  <div className="space-y-6">
                    {revisions.map((rev) => (
                      <div key={rev.id} className="relative pl-10">
                        <div className="absolute left-2.5 top-1 h-3 w-3 rounded-full border-2 border-primary bg-background" />
                        <div className="border rounded-md p-4 space-y-2">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center space-x-2">
                              <span className="font-mono font-bold text-lg">Rev {rev.revision}</span>
                              <Badge variant={rev.status === 'implemented' ? 'default' : rev.status === 'approved' ? 'default' : 'secondary'}>
                                {rev.status}
                              </Badge>
                            </div>
                            {rev.effectivity_date && (
                              <span className="text-sm text-muted-foreground">
                                Effective: {rev.effectivity_date}
                              </span>
                            )}
                          </div>
                          {rev.changes_summary && (
                            <p className="text-sm">{rev.changes_summary}</p>
                          )}
                          <div className="flex flex-wrap gap-4 text-xs text-muted-foreground">
                            <span className="flex items-center gap-1">
                              <User className="h-3 w-3" /> Created by {rev.created_by} on {formatDate(rev.created_at)}
                            </span>
                            {rev.approved_by && rev.approved_at && (
                              <span className="flex items-center gap-1">
                                <CheckCircle className="h-3 w-3 text-green-600" /> Approved by {rev.approved_by} on {formatDate(rev.approved_at)}
                              </span>
                            )}
                            {rev.implemented_by && rev.implemented_at && (
                              <span className="flex items-center gap-1">
                                <Settings className="h-3 w-3 text-blue-600" /> Implemented by {rev.implemented_by} on {formatDate(rev.implemented_at)}
                              </span>
                            )}
                          </div>
                          {rev.notes && (
                            <p className="text-xs text-muted-foreground italic">{rev.notes}</p>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Status Actions */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Settings className="h-5 w-5 mr-2" />
                Actions
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {canApprove && (
                <Button 
                  className="w-full" 
                  onClick={() => handleStatusAction('approve')}
                  disabled={actionLoading === 'approve'}
                >
                  <CheckCircle className="h-4 w-4 mr-2" />
                  {actionLoading === 'approve' ? 'Approving...' : 'Approve ECO'}
                </Button>
              )}
              
              {canImplement && (
                <Button 
                  className="w-full" 
                  onClick={() => handleStatusAction('implement')}
                  disabled={actionLoading === 'implement'}
                >
                  <Settings className="h-4 w-4 mr-2" />
                  {actionLoading === 'implement' ? 'Implementing...' : 'Implement ECO'}
                </Button>
              )}
              
              {canReject && (
                <Button 
                  variant="destructive" 
                  className="w-full" 
                  onClick={() => handleStatusAction('reject')}
                  disabled={actionLoading === 'reject'}
                >
                  <XCircle className="h-4 w-4 mr-2" />
                  {actionLoading === 'reject' ? 'Rejecting...' : 'Reject ECO'}
                </Button>
              )}

              {!canApprove && !canImplement && !canReject && (
                <p className="text-sm text-muted-foreground text-center py-4">
                  No actions available for {statusConfig_.label.toLowerCase()} ECOs
                </p>
              )}
            </CardContent>
          </Card>

          {/* Metadata */}
          <Card>
            <CardHeader>
              <CardTitle>Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center space-x-3">
                <User className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="text-sm font-medium">Created by</p>
                  <p className="text-sm text-muted-foreground">{eco.created_by}</p>
                </div>
              </div>

              <div className="flex items-center space-x-3">
                <Calendar className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="text-sm font-medium">Created</p>
                  <p className="text-sm text-muted-foreground">{formatDate(eco.created_at)}</p>
                </div>
              </div>

              <div className="flex items-center space-x-3">
                <Calendar className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="text-sm font-medium">Last updated</p>
                  <p className="text-sm text-muted-foreground">{formatDate(eco.updated_at)}</p>
                </div>
              </div>

              {eco.approved_at && eco.approved_by && (
                <div className="flex items-center space-x-3">
                  <CheckCircle className="h-4 w-4 text-green-600" />
                  <div>
                    <p className="text-sm font-medium">Approved</p>
                    <p className="text-sm text-muted-foreground">
                      {formatDate(eco.approved_at)} by {eco.approved_by}
                    </p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
export default ECODetail;
