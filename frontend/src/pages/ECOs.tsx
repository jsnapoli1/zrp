import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "../components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "../components/ui/form";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import { Skeleton } from "../components/ui/skeleton";
import { FileText, Plus, Calendar, User } from "lucide-react";
import { api, type ECO } from "../lib/api";
import { useForm } from "react-hook-form";

interface CreateECOData {
  title: string;
  description: string;
  reason: string;
  affected_ipns: string;
}

const statusConfig = {
  draft: { label: 'Draft', variant: 'secondary' as const, color: 'text-gray-600' },
  open: { label: 'Open', variant: 'default' as const, color: 'text-blue-600' },
  approved: { label: 'Approved', variant: 'default' as const, color: 'text-green-600' },
  implemented: { label: 'Implemented', variant: 'outline' as const, color: 'text-green-800' },
  rejected: { label: 'Rejected', variant: 'destructive' as const, color: 'text-red-600' },
};

function ECOs() {
  const navigate = useNavigate();
  const [ecos, setECOs] = useState<ECO[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('all');
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [creating, setCreating] = useState(false);

  const form = useForm<CreateECOData>({
    defaultValues: {
      title: '',
      description: '',
      reason: '',
      affected_ipns: '',
    },
  });

  useEffect(() => {
    fetchECOs();
  }, [activeTab]);

  const fetchECOs = async () => {
    setLoading(true);
    try {
      const statusFilter = activeTab === 'all' ? undefined : activeTab;
      const data = await api.getECOs(statusFilter);
      setECOs(data);
    } catch (error) {
      console.error('Failed to fetch ECOs:', error);
      setECOs([]);
    } finally {
      setLoading(false);
    }
  };

  const handleRowClick = (id: string) => {
    navigate(`/ecos/${id}`);
  };

  const handleCreateECO = async (data: CreateECOData) => {
    setCreating(true);
    try {
      const ecoData = {
        title: data.title,
        description: data.description,
        reason: data.reason,
        affected_ipns: data.affected_ipns,
        status: 'draft',
        priority: 'normal',
      };
      
      const newECO = await api.createECO(ecoData);
      setCreateDialogOpen(false);
      form.reset();
      
      // Navigate to the new ECO detail page
      navigate(`/ecos/${newECO.id}`);
    } catch (error) {
      console.error('Failed to create ECO:', error);
    } finally {
      setCreating(false);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const getStatusConfig = (status: string) => {
    return statusConfig[status as keyof typeof statusConfig] || statusConfig.draft;
  };

  const tabCounts = {
    all: ecos.length,
    open: ecos.filter(eco => ['draft', 'open'].includes(eco.status)).length,
    approved: ecos.filter(eco => eco.status === 'approved').length,
    implemented: ecos.filter(eco => eco.status === 'implemented').length,
    rejected: ecos.filter(eco => eco.status === 'rejected').length,
  };

  const filteredECOs = activeTab === 'all' ? ecos : 
    activeTab === 'open' ? ecos.filter(eco => ['draft', 'open'].includes(eco.status)) :
    ecos.filter(eco => eco.status === activeTab);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Engineering Change Orders</h1>
          <p className="text-muted-foreground">
            Manage design changes and product modifications
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create ECO
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[600px]">
            <Form {...form}>
              <form onSubmit={form.handleSubmit(handleCreateECO)} className="space-y-6">
                <DialogHeader>
                  <DialogTitle>Create New ECO</DialogTitle>
                  <DialogDescription>
                    Create a new Engineering Change Order to document and track modifications.
                  </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                  <FormField
                    control={form.control}
                    name="title"
                    rules={{ required: 'Title is required' }}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Title</FormLabel>
                        <FormControl>
                          <Input placeholder="Enter ECO title..." {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="description"
                    rules={{ required: 'Description is required' }}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Description</FormLabel>
                        <FormControl>
                          <Textarea 
                            placeholder="Describe the change in detail..." 
                            rows={4}
                            {...field} 
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="reason"
                    rules={{ required: 'Reason is required' }}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Reason for Change</FormLabel>
                        <FormControl>
                          <Textarea 
                            placeholder="Why is this change needed?" 
                            rows={3}
                            {...field} 
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="affected_ipns"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Affected IPNs</FormLabel>
                        <FormControl>
                          <Input 
                            placeholder="Comma-separated list of affected part numbers..." 
                            {...field} 
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <DialogFooter>
                  <Button 
                    type="button" 
                    variant="outline" 
                    onClick={() => setCreateDialogOpen(false)}
                    disabled={creating}
                  >
                    Cancel
                  </Button>
                  <Button type="submit" disabled={creating}>
                    {creating ? 'Creating...' : 'Create ECO'}
                  </Button>
                </DialogFooter>
              </form>
            </Form>
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>ECO Status</CardTitle>
        </CardHeader>
        <CardContent>
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="grid w-full grid-cols-5">
              <TabsTrigger value="all">All ({tabCounts.all})</TabsTrigger>
              <TabsTrigger value="open">Open ({tabCounts.open})</TabsTrigger>
              <TabsTrigger value="approved">Approved ({tabCounts.approved})</TabsTrigger>
              <TabsTrigger value="implemented">Implemented ({tabCounts.implemented})</TabsTrigger>
              <TabsTrigger value="rejected">Rejected ({tabCounts.rejected})</TabsTrigger>
            </TabsList>

            <TabsContent value={activeTab} className="mt-6">
              {loading ? (
                <div className="space-y-3">
                  {Array.from({ length: 5 }).map((_, i) => (
                    <Skeleton key={i} className="h-16 w-full" />
                  ))}
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>ECO ID</TableHead>
                      <TableHead>Title</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Created By</TableHead>
                      <TableHead>Created Date</TableHead>
                      <TableHead>Updated Date</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredECOs.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                          {activeTab === 'all' 
                            ? 'No ECOs found' 
                            : `No ${activeTab} ECOs found`
                          }
                        </TableCell>
                      </TableRow>
                    ) : (
                      filteredECOs.map((eco) => {
                        const statusConfig = getStatusConfig(eco.status);
                        return (
                          <TableRow 
                            key={eco.id}
                            className="cursor-pointer hover:bg-muted/50"
                            onClick={() => handleRowClick(eco.id)}
                          >
                            <TableCell>
                              <div className="flex items-center">
                                <FileText className="h-4 w-4 mr-2 text-muted-foreground" />
                                <span className="font-mono font-medium">{eco.id}</span>
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="max-w-sm">
                                <div className="font-medium truncate">{eco.title}</div>
                                {eco.description && (
                                  <div className="text-sm text-muted-foreground truncate">
                                    {eco.description}
                                  </div>
                                )}
                              </div>
                            </TableCell>
                            <TableCell>
                              <Badge variant={statusConfig.variant} className={statusConfig.color}>
                                {statusConfig.label}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center">
                                <User className="h-4 w-4 mr-2 text-muted-foreground" />
                                {eco.created_by}
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center">
                                <Calendar className="h-4 w-4 mr-2 text-muted-foreground" />
                                {formatDate(eco.created_at)}
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center">
                                <Calendar className="h-4 w-4 mr-2 text-muted-foreground" />
                                {formatDate(eco.updated_at)}
                              </div>
                            </TableCell>
                          </TableRow>
                        );
                      })
                    )}
                  </TableBody>
                </Table>
              )}
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  );
}
export default ECOs;
