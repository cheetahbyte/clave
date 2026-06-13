import { Check, ChevronDown, Package, Plus, Image as ImageIcon } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { useCurrentProduct } from "@/features/admin/product-context";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";

export function ProductSwitcher() {
  const { productId, product, products, setProductId } = useCurrentProduct();

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <div className="bg-sidebar-accent text-sidebar-accent-foreground flex aspect-square size-8 items-center justify-center overflow-hidden rounded-lg border">
                {product?.logoUrl ? (
                  <img src={product.logoUrl} alt="" className="size-full object-contain" />
                ) : (
                  <Package className="size-4" />
                )}
              </div>
              <span className="flex-1 truncate text-sm font-semibold">{product?.name ?? "No product"}</span>
              <ChevronDown className="size-4 opacity-50" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-full min-w-[220px]"
            align="start"
            side="bottom"
            sideOffset={4}
          >
            <DropdownMenuLabel className="text-muted-foreground text-xs">Products</DropdownMenuLabel>
            {products.length === 0 ? (
              <DropdownMenuItem disabled className="text-muted-foreground gap-2 p-2 text-xs">
                No products yet
              </DropdownMenuItem>
            ) : (
              products.map((p) => (
                <DropdownMenuItem
                  key={p.id}
                  className="gap-2 p-2"
                  onClick={() => setProductId(p.id)}
                >
                  <div className="flex size-6 items-center justify-center overflow-hidden rounded-md border">
                    {p.logoUrl ? (
                      <img src={p.logoUrl} alt="" className="size-full object-contain" />
                    ) : (
                      <ImageIcon className="size-3.5 shrink-0" />
                    )}
                  </div>
                  <span className="flex-1 truncate">{p.name}</span>
                  {p.id === productId ? <Check className="size-4" /> : null}
                </DropdownMenuItem>
              ))
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild className="gap-2 p-2">
              <Link to="/products">
                <div className="flex size-6 items-center justify-center rounded-md border bg-transparent">
                  <Plus className="size-4" />
                </div>
                <div className="text-muted-foreground font-medium">Manage products</div>
              </Link>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
