#!/usr/local/bin/gentee

// Copyright 2019 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

func gcd( int left right ) int
{
    if right == 0 : return left
    return gcd( right, left % right )
}

run  {
    str     input
    int     left right
    
    Println("This program finds the greatest common divisor by the Euclidean Algorithm.")
 
    while true
    {
       left = int( ReadString( "Enter the first number ( enter 0 to exit ): "))
       if left == 0 : break
       
       right = int( ReadString( "Enter the second number: "))
       Println("GCD = \{ gcd( left, right )}")
    }
}