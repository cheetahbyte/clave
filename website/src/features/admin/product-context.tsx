import { createContext, useContext, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { listAdminProducts, type AdminProductItem } from "@/features/admin/api";

const STORAGE_KEY = "clave.currentProductId";

type ProductContextValue = {
  productId: string | null;
  product: AdminProductItem | null;
  products: AdminProductItem[];
  isLoading: boolean;
  setProductId: (id: string) => void;
};

const ProductContext = createContext<ProductContextValue | null>(null);

export function ProductProvider({ children }: { children: React.ReactNode }) {
  const { data: products, isLoading } = useQuery({
    queryKey: ["adminProducts"],
    queryFn: listAdminProducts,
  });

  const [storedId, setStoredId] = useState<string | null>(() => {
    try {
      return localStorage.getItem(STORAGE_KEY);
    } catch {
      return null;
    }
  });

  function setProductId(id: string) {
    setStoredId(id);
    try {
      localStorage.setItem(STORAGE_KEY, id);
    } catch {
      // ignore (private mode etc.)
    }
  }

  // Resolve the effective product id: keep the stored one if it still exists
  // for this org, otherwise fall back to the first product (or null).
  const list = products ?? [];
  const productId = useMemo(() => {
    if (storedId && list.some((p) => p.id === storedId)) return storedId;
    return list[0]?.id ?? null;
  }, [storedId, list]);

  // Persist the fallback so the rest of the app and a refresh agree.
  useEffect(() => {
    if (productId && productId !== storedId) {
      setProductId(productId);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [productId]);

  const product = list.find((p) => p.id === productId) ?? null;

  const value: ProductContextValue = {
    productId,
    product,
    products: list,
    isLoading,
    setProductId,
  };

  return <ProductContext.Provider value={value}>{children}</ProductContext.Provider>;
}

export function useCurrentProduct(): ProductContextValue {
  const ctx = useContext(ProductContext);
  if (!ctx) throw new Error("useCurrentProduct must be used within a ProductProvider");
  return ctx;
}
