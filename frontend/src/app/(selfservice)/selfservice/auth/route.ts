import { validateToken } from "@/actions/selfservice";
import { NextResponse } from "next/server";

interface RouteContext {
  params: {
    token: string;
  };
}

export async function GET(
  request: Request,
  { params }: RouteContext
) {
  const { searchParams } = new URL(request.url);

  const token = searchParams.get("token");
  if (!token) {
    return NextResponse.json(
      { message: "Missing token" },
      { status: 400 }
    );
  }

  const jwt = await validateToken(token);

  if (!jwt) {
    return NextResponse.json(
      { message: "Invalid token" },
      { status: 401 }
    );
  }

  const response = NextResponse.redirect("http://localhost:3000/selfservice");

  response.cookies.set({
    name: "selfservice_session",
    value: jwt,
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24, // 1 day
  });

  return response;
}
